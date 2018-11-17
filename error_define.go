// 错误定义

package apiinit

import (
	"github.com/go-apibox/api"
)

// error type
const (
	errorModuleInitWaiting = iota
)

var ErrorDefines = map[api.ErrorType]*api.ErrorDefine{
	errorModuleInitWaiting: api.NewErrorDefine(
		"ModuleInitWaiting",
		[]int{0},
		map[string]map[int]string{
			"en_us": {
				0: "Module is waiting for initialization!",
			},
			"zh_cn": {
				0: "模块正在等待初始化！",
			},
		},
	),
}
