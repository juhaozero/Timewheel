# Timewheel
Golang实现的时间轮



# 原理
 利用了链表来把执行任务压入队尾，用了一个大定时器去间隔时间移动指针，读取双向链表内的任务,任务上有circle来表示圈数及分层
 单时间轮有多个任务，所以利用工作池进行并发，当circle为1表示为当前层，压入工作池,删除该任务,当持续次数为-1时,持续加入链表,实现循环,否则持续时间减1。



#### 部分想法来自于，感谢
https://www.luozhiyun.com/archives/444

![时间轮](https://ask.qcloudimg.com/http-save/2108867/npg7wbu80w.png?imageView2/2/w/1620)

# 使用

```go
func main() {
	// 初始化一个时间轮对象
	timeWheel := NewTimeWheel()

添加循环的方法	// 简单的示例

	// 添加方法
    // 第一个参数为执行的时间
    // 第二个参数为执行的次数
    // 第二个参数为执行的函数
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
