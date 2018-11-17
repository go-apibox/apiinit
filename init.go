package apiinit

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-apibox/api"
	"github.com/go-apibox/apiproxy"
	"github.com/go-apibox/apisign"
)

type Init struct {
	app           *api.App
	disabled      bool
	inited        bool
	callbackFuncs []func()
}

func NewInit(app *api.App) *Init {
	app.Error.RegisterGroupErrors("init", ErrorDefines)

	cfg := app.Config
	disabled := cfg.GetDefaultBool("apiinit.disabled", false)
	if disabled {
		return &Init{app, true, false, []func(){}}
	}
	return &Init{app, disabled, false, []func(){}}
}

func (bm *Init) AddCallback(callbackFunc func()) {
	bm.callbackFuncs = append(bm.callbackFuncs, callbackFunc)
}

func (bm *Init) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if bm.disabled {
		next(w, r)
		return
	}
	if bm.inited {
		// 已初始化，则直接跳过
		next(w, r)
		return
	}

	c, err := api.NewContext(bm.app, w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	action := c.Input.GetAction()
	if action == "APIBox.Init" {
		// 处理参数中的 [中间件].[设置项]
		params := c.Input.GetAll()

		// 将参数按中间件分组
		signParams := make(map[string]string)
		proxyParams := make(map[string]map[string]string)
		for k, v := range params {
			kFields := strings.SplitN(k, ".", 2)
			if len(kFields) != 2 {
				continue
			}
			switch kFields[0] {
			case "apisign":
				signParams[kFields[1]] = v
			case "apiproxy":
				tFields := strings.SplitN(kFields[1], ".", 2)
				if len(tFields) != 2 {
					continue
				}
				backendAlias := tFields[0]
				if _, ok := proxyParams[backendAlias]; !ok {
					proxyParams[backendAlias] = make(map[string]string)
				}
				proxyParams[backendAlias][tFields[1]] = v
			}
		}

		if len(signParams) > 0 {
			if obj, has := bm.app.Middlewares["apisign"]; has {
				sign := obj.(*apisign.Sign)
				for k, v := range signParams {
					switch k {
					case "sign_key":
						sign.SetSignKey(v)
					case "disabled":
						if v == "N" {
							sign.Enable()
						} else {
							sign.Disable()
						}
					}
				}
			}
		}

		if len(proxyParams) > 0 {
			if obj, has := bm.app.Middlewares["apiproxy"]; has {
				proxy := obj.(*apiproxy.Proxy)
				for backendAlias, params := range proxyParams {
					client := proxy.GetClient(backendAlias)
					if client == nil {
						continue
					}
					for k, v := range params {
						switch k {
						case "gwurl":
							client.GWURL = v
						case "gwaddr":
							client.GWADDR = v
						case "appid":
							client.AppId = v
						case "sign_key":
							client.SignKey = v
						case "nonce_enabled":
							if v == "Y" {
								client.NonceEnabled = true
							} else {
								client.NonceEnabled = false
							}
						case "nonce_length":
							client.NonceLength, _ = strconv.Atoi(v)
						default:
							if strings.IndexByte(k, '.') != -1 {
								tFields := strings.SplitN(k, ".", 2)
								if len(tFields) == 2 {
									switch tFields[0] {
									case "default_params":
										client.SetDefaultParam(tFields[1], v)
									case "override_params":
										client.SetOverrideParam(tFields[1], v)
									}
								}
							}
						}
					}
				}
			}
		}

		bm.inited = true
		api.WriteResponse(c, nil)

		go func() {
			for _, callbackFunc := range bm.callbackFuncs {
				callbackFunc()
			}
		}()
		return
	}

	if !bm.inited {
		api.WriteResponse(c, bm.app.Error.NewGroupError("init", errorModuleInitWaiting))
		return
	}
	next(w, r)
}
