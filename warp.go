package dispatch

import (
	"runtime"
	"sync"

	"github.com/azer/logger"
)

var log = logger.New("dispatch")

type Waiter struct {
	sync.WaitGroup
}

func (w *Waiter) Warp(f func()) {
	w.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("waiter recover a error:%v", r)
				size := 1 << 18
				buf := make([]byte, size)
				n := runtime.Stack(buf, false)
				log.Error("----------------Panic Stack---------------: \n%v\n", string(buf[:n]))
				//maybe send a msg to oncall
			}
			w.Done()
		}()

		f()
	}()
}
