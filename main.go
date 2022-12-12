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

	// 添加循环的方法
	timeWheel.AddTimer(1*time.Second, -1, func() {
		fmt.Println("====时间轮走到了该处==== 执行循环")
	})

}
func NewTimeWheel() *timer.TimeWheel {
	// 第一个参数为tick刻度, 即时间轮多久转动一次
	// 第二个参数为时间轮槽slot数量
	// 第三个参数为工作池的数量
	t, err := timer.NewTimeWheel(50*time.Millisecond, 1000, 3)
	if err != nil {
		panic(err)
	}
	t.Start()
	return t
}
