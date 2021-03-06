package mapreduce

import (
	"fmt"
	"sync"
)

//
// schedule() 会启动并等待指定阶段（Map 或 Reduce）的所有任务完成。mapFiles
// 参数包含所有用作 Map 阶段输入文件的名称，每个对应一个 Map 任务。nReduce
// 是 Reduce 任务的数量。registerChan 参数会提供一个包含已注册 Worker
// 的 Channel，每个元素都是 Worker 的 RPC 地址，可被传入到 call() 函数中。
// registerChan 会返回所有已注册的 Worker，并在后续有新的 Worker 注册时
// 返回新的 Worker
//
func schedule(jobName string, mapFiles []string, nReduce int, phase jobPhase, registerChan chan string) {
	var nTasks int
	var nOther int // Reduce 阶段的输入文件数量或 Map 阶段的输出文件数量
	switch phase {
	case mapPhase:
		nTasks = len(mapFiles)
		nOther = nReduce
	case reducePhase:
		nTasks = nReduce
		nOther = len(mapFiles)
	}

	fmt.Printf("Schedule: %v %v tasks (%d I/Os)\n", nTasks, phase, nOther)

	// 所有 nTasks 个任务会要被调度到 Worker 上，并且它们都成功完成后函数必须立刻返回。
	// 记住，Worker 有可能会失效，而且任意 Worker 可能会完成多个任务。
	//
	// TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO
	//

	// !!! 以下是 Mr-Dai 的参考实现 !!!

	// 初始化待完成任务池
	tasks := make(chan int, nTasks)
	for i := 0; i < nTasks; i++ {
		tasks <- i
	}
	wg := sync.WaitGroup{}
	wg.Add(nTasks)

	done := make(chan struct{})
	// 启动主 Goroutine，接收新 Worker 的注册事件
	go func() {
		for {
			select {
			case <-done:
				break
			case wk := <-registerChan:
				// 为新注册的 Worker 启动调度 Goroutine
				go func(wk string) {
					for {
						select {
						case <-done:
							break
						case task := <-tasks:
							// 从任务池中获取一个待完成的任务并调度运行
							args := DoTaskArgs{JobName: jobName, Phase: phase, TaskNumber: task, NumOtherPhase: nOther}
							if phase == mapPhase {
								args.File = mapFiles[task]
							}

							ok := call(wk, "Worker.DoTask", args, nil)
							if !ok {
								// 任务运行失败，重新回到任务池
								debug("schedule: worker %s %s task #%d - failed\n", wk, phase, task)
								tasks <- task
							} else {
								debug("schedule: worker %s %s task #%d - succeeded\n", wk, phase, task)
								wg.Done()
							}
						}
					}
				}(wk)
			}
		}
	}()

	// 等待所有任务执行完成后通知所有 Goroutine 退出
	wg.Wait()
	close(done)
	fmt.Printf("Schedule: %v phase done\n", phase)
}
