package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Coordinator struct {
	// Your definitions here.
	mut sync.Mutex

	filename []string
	nreduce  int

	maptask    []Task
	reducetask []Task

	maptaskfinish    bool
	reducetaskfinish bool
}

// Your code here -- RPC handlers for the worker to call.

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

// 分配Map任务给Worker
func (c *Coordinator) AssignMission(args *CallArgs, reply *TaskReply) error {
	c.mut.Lock()
	defer c.mut.Unlock()
	tmpCheckAllFinish := true // 检测是否全完成了。如果是false说明至少有一个任务ready或running
	// 首先完成map任务分配
	if !c.maptaskfinish {
		i := 0
		for ; i < len(c.maptask); i++ {
			currenttime := time.Now()

			// 如果任务未分配或者任务超过了10秒
			if (c.maptask[i].missionstate == Ready) || (c.maptask[i].missionstate == Running && currenttime.Sub(c.maptask[i].timestamp).Seconds() > 10) {
				// 重新分配
				c.maptask[i].timestamp = time.Now() //更新时间戳
				c.maptask[i].missionstate = Running
				reply.task = c.maptask[i]

				// 分配任务成功，说明map任务未完成
				c.maptaskfinish = false

				// 完成，返回给worker执行
				return nil
			} else if c.maptask[i].missionstate == Running {
				// 还有任务正在运行，map任务没有结束
				tmpCheckAllFinish = false
			}
		}
		// 遍历了一遍，可以确认是否完成了
		c.maptaskfinish = tmpCheckAllFinish

		//如果运行到这里，说明没有找到一个合适的任务进行分配。因此分配为waiting
		assert(i == len(c.maptask), "任务分配出错：未遍历完成任务列表即跳出")
		reply.task.worktype = Waiting
		return nil
	} else {
		// 完成reduce任务分配
	}

	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	ret := false

	// Your code here.

	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}
	// 第一步，初始化C的数据结构
	// 为C设定nReduce个数
	c.nreduce = nReduce

	//初始化要分配的文件名
	for _, possiblefile := range files {
		files, err := filepath.Glob(possiblefile)
		if err != nil {
			fmt.Println("错误读取文件：", err)
			return nil
		}
		for _, filename := range files {
			c.filename = append(c.filename, filename)
		}
	}

	//初始化可以分配的maptask
	for i, filename := range c.filename {
		tmp := Task{}
		tmp.missionstate = Ready
		tmp.index = i
		tmp.mapfile = filename
		c.maptask = append(c.maptask, tmp)

	}

	// 初始化reducetask
	for i := 0; i < c.nreduce; i++ {
		c.reducetask = append(c.reducetask, Task{})
		c.reducetask[i].worktype = Reduce
		c.reducetask[i].index = i
		c.reducetask[i].missionstate = Ready

	}

	c.server()
	return &c
}