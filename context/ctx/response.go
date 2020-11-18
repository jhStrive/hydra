package ctx

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/clbanning/mxj"
	"github.com/micro-plat/hydra/conf"
	"github.com/micro-plat/hydra/conf/app"
	"github.com/micro-plat/hydra/context"
	"github.com/micro-plat/hydra/global"
	"github.com/micro-plat/lib4go/encoding"
	"github.com/micro-plat/lib4go/errs"
	"github.com/micro-plat/lib4go/logger"
	"github.com/micro-plat/lib4go/types"
	"gopkg.in/yaml.v2"
)

var _ context.IResponse = &response{}

type rspns struct {
	status      int
	contentType string
	content     interface{}
}

type response struct {
	ctx         context.IInnerContext
	conf        app.IAPPConf
	path        *rpath
	raw         rspns
	final       rspns
	hasWrite    bool
	noneedWrite bool
	log         logger.ILogger
	asyncWrite  func() error
	specials    []string
}

func NewResponse(ctx context.IInnerContext, conf app.IAPPConf, log logger.ILogger, meta conf.IMeta) *response {
	return &response{
		ctx:  ctx,
		conf: conf,
		path: NewRpath(ctx, conf, meta),
		log:  log,
	}
}

//Header 设置头信息到response里
func (c *response) Header(k string, v string) {
	c.ctx.Header(k, v)
}

//Header 设置头信息到response里
func (c *response) GetHeaders() map[string][]string {
	return c.ctx.GetHeaders()
}

//ContentType 设置contentType
func (c *response) ContentType(v string) {
	c.ctx.Header("Content-Type", v)
}

//Abort 设置错误码与错误消息,并将数据写入响应流,并终止应用
func (c *response) Abort(s int, errs ...error) {
	defer c.ctx.Abort()
	defer c.Flush()
	if len(errs) > 0 {
		c.Write(s, errs[0])
		return
	}
	c.Write(s)
}

//File 将指定的文件的返回,文件数据写入响应流,并终止应用
func (c *response) File(path string) {
	defer c.ctx.Abort()
	if c.noneedWrite || c.ctx.Written() {
		return
	}
	c.noneedWrite = true
	c.ctx.WStatus(http.StatusOK)
	c.ctx.File(path)
}

//NoNeedWrite 无需写入响应数据到缓存
func (c *response) NoNeedWrite(status int) {
	c.noneedWrite = true
	c.final.status = status
}

//Write 检查内容并处理状态码,数据未写入响应流
func (c *response) Write(status int, ct ...interface{}) error {
	if c.noneedWrite {
		return fmt.Errorf("不能重复写入到响应流:status:%d 已写入状态:%d", status, c.final.status)
	}

	//1. 处理content
	var content interface{}
	if len(ct) > 0 {
		content = ct[0]
	}

	//2. 修改当前结果状态码与内容
	var ncontent interface{}
	c.final.status, ncontent = c.swapBytp(status, content)
	c.final.contentType, c.final.content = c.swapByctp(ncontent)
	if strings.Contains(c.final.contentType, "%s") {
		c.final.contentType = fmt.Sprintf(c.final.contentType, c.path.GetEncoding())
	}
	if c.hasWrite {
		return nil
	}

	//3. 保存初始状态与结果
	c.raw.status, c.raw.content, c.hasWrite, c.raw.contentType = status, content, true, c.final.contentType
	c.asyncWrite = func() error {
		return c.writeNow(c.final.status, c.final.contentType, c.final.content.(string))
	}
	return nil
}

//WriteAny 向响应流中写入内容,状态码根据内容进行判断(不会立即写入)
func (c *response) WriteAny(v interface{}) error {
	return c.Write(http.StatusOK, v)
}
func (c *response) swapBytp(status int, content interface{}) (rs int, rc interface{}) {
	rs = status
	rc = content
	if content == nil {
		rc = ""
	}
	if status == 0 {
		rs = http.StatusOK
	}
	switch v := content.(type) {
	case errs.IError:
		rs = v.GetCode()
		rc = v.GetError().Error()
		c.log.Error(content)
		if !global.IsDebug {
			rc = "Internal Server Error"
		}
	case error:
		if status >= http.StatusOK && status < http.StatusBadRequest {
			rs = http.StatusBadRequest
		}
		c.log.Error(content)
		rc = v.Error()
		if !global.IsDebug {
			rc = "Internal Server Error"
		}
	default:
		return rs, rc
	}
	return rs, rc
}

func (c *response) swapByctp(content interface{}) (string, string) {
	fmt.Println("content:", content)
	ctp := c.getContentType()
	switch {
	case strings.Contains(ctp, "plain"):
		return ctp, fmt.Sprint(content)
	default:
		if content == nil || content == "" {
			return types.GetString(ctp, context.PLAINF), ""
		}
		tp := reflect.TypeOf(content).Kind()
		value := reflect.ValueOf(content)
		if tp == reflect.Ptr {
			value = value.Elem()
		}
		switch tp {
		case reflect.String:
			text := []byte(fmt.Sprint(content))
			switch {
			case (ctp == "" || strings.Contains(ctp, "json")) && json.Valid(text) && (bytes.HasPrefix(text, []byte("{")) ||
				bytes.HasPrefix(text, []byte("["))):
				return context.JSONF, content.(string)
			case strings.Contains(ctp, "html") && bytes.HasPrefix(text, []byte("<!DOCTYPE html")):
				return context.HTMLF, content.(string)
			case strings.Contains(ctp, "yaml"):
				return context.YAMLF, content.(string)
			case ctp == "" || strings.Contains(ctp, "plain"):
				return context.PLAINF, content.(string)
			default:
				_, errx := mxj.BeautifyXml(text, "", "")
				if errx != io.EOF {
					return context.XMLF, content.(string)
				}

				return ctp, c.getString(ctp, map[string]interface{}{
					"data": content,
				})
			}
		case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
			if ctp == "" {
				return context.PLAINF, fmt.Sprint(content)
			}
			return ctp, c.getString(ctp, map[string]interface{}{
				"data": content,
			})
		default:
			if ctp == "" {
				c.ContentType("application/json; charset=UTF-8")
				return context.JSONF, c.getString(context.JSONF, content)
			}
			return ctp, c.getString(ctp, content)
		}

	}
}

func (c *response) getContentType() string {
	if ctp := c.ctx.WHeader("Content-Type"); ctp != "" {
		return ctp
	}
	headerObj, err := c.conf.GetHeaderConf()
	if err != nil {
		return ""
	}
	if ct, ok := headerObj["Content-Type"]; ok && ct != "" {
		return ct
	}
	return ""
}

//writeNow 将状态码、内容写入到响应流中
func (c *response) writeNow(status int, ctyp string, content string) error {
	//301 302 303 307 308 这个地方会强制跳转到content 的路径。
	if status == http.StatusMovedPermanently || status == http.StatusFound || status == http.StatusSeeOther ||
		status == http.StatusTemporaryRedirect || status == http.StatusPermanentRedirect {
		//从header里面获取的Location
		location := content
		if l := c.ctx.WHeader("Location"); l != "" {
			location = l
		}
		c.ctx.Redirect(status, location)
		return nil
	}

	buff := []byte(content)
	e := c.path.GetEncoding()
	var err error

	if e != "utf-8" {
		buff, err = encoding.Encode(content, e)
		if err != nil {
			return fmt.Errorf("输出时进行%s编码转换错误：%w %s", e, err, content)
		}
	}

	c.ctx.Data(status, ctyp, buff)
	return nil
}

//Redirect 转跳g刚才gc
func (c *response) Redirect(code int, url string) {
	c.ctx.Redirect(code, url)
}

//AddSpecial 添加响应的特殊字符
func (c *response) AddSpecial(t string) {
	if c.specials == nil {
		c.specials = make([]string, 0, 1)
	}
	c.specials = append(c.specials, t)
}

//GetSpecials 获取多个响应特殊字符
func (c *response) GetSpecials() string {
	return strings.Join(c.specials, "|")
}

//GetRaw 获取原始响应请求
func (c *response) GetRaw() interface{} {
	return c.raw.content
}

//GetRawResponse 获取响应内容信息
func (c *response) GetRawResponse() (int, interface{}) {
	return c.raw.status, c.raw.content
}

//GetFinalResponse 获取响应内容信息
func (c *response) GetFinalResponse() (int, string) {
	if c.final.content == nil {
		return c.final.status, ""
	}
	return c.final.status, c.final.content.(string)
}

//Flush 调用异步写入将状态码、内容写入到响应流中
func (c *response) Flush() {
	if c.noneedWrite || c.asyncWrite == nil || c.ctx.Written() {
		return
	}
	if err := c.asyncWrite(); err != nil {
		panic(err)
	}
	//@fix 放在异步之前中间件recovery不能重写
	c.noneedWrite = true
}

func (c *response) getString(ctp string, v interface{}) string {
	switch {
	case strings.Contains(ctp, "xml"):
		tp := reflect.TypeOf(v).Kind()
		if tp == reflect.Map {
			if s, ok := v.(map[string]interface{}); ok { //@fix 修改成的xml转为多层的map
				m := mxj.New()
				m = s
				str, err := m.Xml()
				if err != nil {
					panic(err)
				}
				return string(str)
			}
		}

		buff, err := xml.Marshal(v)
		if err != nil {
			panic(err)
		}
		return string(buff)
	case strings.Contains(ctp, "yaml"):
		buff, err := yaml.Marshal(v)
		if err != nil {
			panic(err)
		}
		return string(buff)
	case strings.Contains(ctp, "json"):
		buff, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		return string(buff)
	}
	return fmt.Sprint(v)
}

func (c *response) getContent() string {
	return c.final.content.(string)
}
func (c *response) getStatus() int {
	return c.raw.status
}
