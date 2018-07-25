package dispatch

import (
	"context"
	"testing"
	"time"
)

type task string

func newTask(s string) task {
	return task(s)
}

func (t task) Name() string           { return string(t) }
func (task) LoadData(context.Context) {}
func (task) IsLoad() bool             { return false }

func TestTaskNode(t *testing.T) {
	n := getTestNodes()
	t.Logf("======= Root Node")
	t.Logf("root --> %v\n", n)
	t.Logf("\n\n")
	t.Log("========Test All Nodes ")
	for i, node := range n.GetAllSubNodes() {
		t.Logf("SubNode %v -- %v\n", i, node)
	}
	t.Logf("\n\n")
	t.Log("========Test Get Nodes ")
	t.Logf("GetNodeByName -- %v\n", n.GetNodeByName("a"))
	t.Logf("GetNodeByName -- %v\n", n.GetNodeByName("f"))
	t.Logf("GetNodeByName -- %v\n", n.GetNodeByName("3"))
	t.Logf("GetNodeByName -- %v\n", n.GetNodeByName("e"))
}

func TestLoadData(t *testing.T) {
	n := getTestNodes()
	t.Logf("root --> %v\n", n)
	t.Log("========Test Node LoadData")
	n.LoadData(context.TODO())
}

func TestPath(t *testing.T) {
	n := getTestNodes()
	t.Logf("root --> %v\n", n)
	t.Logf("======= Test All Path")
	nodes := n.GetAllPath()
	t.Logf("Path Count: %v\n", len(nodes))
	for i, _ := range nodes {
		t.Logf("Path %v:  %v\n", i, pathName(nodes[i]))
	}
	t.Logf("\n\n")
}

// a->b->c->e
// a->d->e
// a->f
// a->x //实际是e

func getTestNodes() *TaskNode {
	a := newTask("a")
	b := newTask("b")
	c := newTask("c")
	d := newTask("d")
	e := newTask("e")
	f := newTask("f")
	x := newTask("x")
	aNode := NewTaskNode(a)
	bNode := NewTaskNode(b)
	cNode := NewTaskNode(b)
	cNode.SetJob(c)
	dNode := NewTaskNode(d)
	eNode := NewTaskNode(e)
	fNode := NewTaskNode(f)
	aNode.AddSubNode(bNode)
	aNode.AddSubNode(dNode)
	aNode.AddSubNode(fNode)
	bNode.AddSubNode(cNode)
	cNode.AddSubNode(eNode)
	dNode.AddSubNode(eNode)
	xNode := NewTaskNode(x)
	aNode.AddSubNode(xNode)
	return aNode
}

func TestMain(m *testing.M) {
	m.Run()
	time.Sleep(5)
}
