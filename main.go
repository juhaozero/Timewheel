package main

import (
	"fmt"
	"time"
	"timewheel/timer"
)

func main() {
	// 初始化一个时间轮对象
	timeWheel := NewTimeWheel()

	// 简单的示例

	// 添加方法
	tid := timeWheel.AddTimer(1*time.Second, 1, func() {
		fmt.Println("====时间轮走到了该处==== 执行1次")
	})
	// 删除方法
	timeWheel.RemoveTimer(tid)

	// 删除方法
	timeWheel.AddTimer(1*time.Second, -1, func() {
		fmt.Println("====时间轮走到了该处==== 执行循环")
	})

}
func NewTimeWheel() *timer.TimeWheel {
	t, err := timer.NewTimeWheel(50*time.Millisecond, 1000, 3)
	if err != nil {
		panic(err)
	}
	t.Start()
	return t
}
