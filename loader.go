package dispatch

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
)

type Job interface {
	Name() string
	LoadData(context.Context)
}

type TaskNode struct {
	job          Job
	subNodes     []*TaskNode
	subNodeMap   map[*TaskNode]struct{}
	preNodes     []*TaskNode
	preNodeMap   map[*TaskNode]struct{}
	isLoad       bool
	taskLock     sync.Mutex
	onceLoadData sync.Once
	onceLoadTask sync.Once
	waiter       Waiter
}

func NewTaskNode(job Job) *TaskNode {
	node := new(TaskNode)
	node.job = job
	return node
}

func (t *TaskNode) LoadData(ctx context.Context) {
	paths := t.GetAllPath()
	if len(paths) == 0 {
		return
	}
	t.onceLoadData.Do(func() {
		for i, _ := range paths {
			path := paths[i]
			level := i
			log.Info("Load Path %v: %v", level, pathName(path))
			t.waiter.Warp(func() {
				for _, node := range path {
					node.TaskLoad(ctx)
				}
			})
		}
		t.waiter.Wait()
	})
}

func (t *TaskNode) LoadDataWithContext(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		defer func() {
			close(done)
		}()
		t.LoadData(ctx)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		log.Error("load data error:%v", ctx.Err())
	}
}

func (t *TaskNode) GetNextNodes() []*TaskNode {
	t.taskLock.Lock()
	nodes := t.subNodes
	t.taskLock.Unlock()
	return nodes
}

func (t *TaskNode) String() string {
	return fmt.Sprintf("jobName:%v subNodes:%v", t.job.Name(), t.subNodes)
}

func (t *TaskNode) IsLoad() bool {
	t.taskLock.Lock()
	isLoad := t.isLoad
	t.taskLock.Unlock()
	return isLoad
}

func (t *TaskNode) SetLoad() {
	t.taskLock.Lock()
	t.isLoad = true
	t.taskLock.Unlock()
}

func (t *TaskNode) Name() string {
	if reflect.ValueOf(t.GetJob()).IsNil() {
		return "unkown"
	}
	return t.GetJob().Name()
}

func (t *TaskNode) TaskLoad(ctx context.Context) {
	t.onceLoadTask.Do(func() {
		t.SetLoad()
		j := t.GetJob()
		if reflect.ValueOf(j).IsNil() {
			return
		}
		j.LoadData(ctx)
	})
}

func (t *TaskNode) GetAllPath() [][]*TaskNode {
	return t.ToNodePath(nil)
}

func (t *TaskNode) ToNodePath(n *TaskNode) [][]*TaskNode {
	paths := make([][]*TaskNode, 0, 10)
	var path = []*TaskNode{t}
	traceNode(t, n, path, &paths)
	return paths
}

func (t *TaskNode) GetPreNodes() []*TaskNode {
	t.taskLock.Lock()
	nodes := t.preNodes
	t.taskLock.Unlock()
	return nodes
}

func (t *TaskNode) GetJob() Job {
	t.taskLock.Lock()
	job := t.job
	t.taskLock.Unlock()
	return job
}

func (t *TaskNode) GetNodeByName(name string) []*TaskNode {
	nodeMap := make(map[*TaskNode]struct{})
	subNodes := t.GetAllPath()
	f := func(t *TaskNode) bool {
		_, find := nodeMap[t]
		nodeMap[t] = struct{}{}
		return !find && t.GetJob().Name() == name
	}
	return rangeNodes(f, subNodes)
}

func (t *TaskNode) SetJob(job Job) {
	t.taskLock.Lock()
	t.job = job
	t.taskLock.Unlock()
}

func (t *TaskNode) addPreNode(node *TaskNode) {
	t.taskLock.Lock()
	if t.preNodeMap == nil {
		t.preNodeMap = make(map[*TaskNode]struct{})
	}
	if t.preNodes == nil {
		t.preNodes = make([]*TaskNode, 0, 0)
	}
	if _, find := t.preNodeMap[node]; !find {
		t.preNodeMap[node] = struct{}{}
		t.preNodes = append(t.preNodes, node)
	}
	t.taskLock.Unlock()
}

//return next node
func (t *TaskNode) Next(node *TaskNode) *TaskNode {
	t.AddSubNode(node)
	return node
}

//return current node
func (t *TaskNode) AddSubNode(nodes ...*TaskNode) *TaskNode {
	t.taskLock.Lock()
	for _, node := range nodes {
		if t.subNodeMap == nil {
			t.subNodeMap = make(map[*TaskNode]struct{})
		}
		if t.subNodes == nil {
			t.subNodes = make([]*TaskNode, 0, 0)
		}
		if _, find := t.subNodeMap[node]; !find {
			t.subNodes = append(t.subNodes, node)
			t.subNodeMap[node] = struct{}{}
		}
		node.addPreNode(t)
	}
	t.taskLock.Unlock()
	return t
}

func (t *TaskNode) GetAllSubNodes() []*TaskNode {
	nodeMap := make(map[*TaskNode]struct{})
	subNodes := t.GetAllPath()
	f := func(t *TaskNode) bool {
		_, find := nodeMap[t]
		nodeMap[t] = struct{}{}
		return !find
	}
	return rangeNodes(f, subNodes)
}

func rangeNodes(f func(t *TaskNode) bool, nodes [][]*TaskNode) []*TaskNode {
	ret := make([]*TaskNode, 0, len(nodes))
	for i, _ := range nodes {
		for j, _ := range nodes[i] {
			if f(nodes[i][j]) {
				ret = append(ret, nodes[i][j])
			}
		}
	}
	return ret
}

func traceNode(aNode *TaskNode, bNode *TaskNode, path []*TaskNode, paths *[][]*TaskNode) {
	aNextNodes := aNode.GetNextNodes()
	if aNextNodes == nil {
		*paths = append(*paths, path)
	}
	for i, _ := range aNextNodes {
		var tempPath = make([]*TaskNode, 0, len(path)+1)
		tempPath = append(tempPath, path...)
		tempPath = append(tempPath, aNextNodes[i])
		traceNode(aNextNodes[i], bNode, tempPath, paths)
		path = tempPath[:len(tempPath)-1]
	}
}

func pathName(path []*TaskNode) string {
	pathNames := make([]string, 0, len(path))
	if len(path) == 0 {
		return "[empty path]"
	}
	for _, node := range path {
		pathNames = append(pathNames, node.Name())
	}
	return fmt.Sprintf("[%v]", strings.Join(pathNames, "-->"))
}

func showPathNames(paths [][]*TaskNode) {
	for i, _ := range paths {
		fmt.Printf("Path %v --: %v, %p", i, pathName(paths[i]), &paths[i])
		fmt.Println()
	}
}

func ShowNode(n *TaskNode) {
	if n == nil {
		fmt.Println("empty node")
		return
	}
	paths := n.GetAllPath()
	showPathNames(paths)
}

func ShowAllNodes(n *TaskNode) {
	nodes := n.GetAllSubNodes()
	names := make([]string, 0, len(nodes))
	for _, node := range nodes {
		names = append(names, node.Name())
	}
	sort.Strings(names)
	log.Info("Nodes:%v", names)
}
