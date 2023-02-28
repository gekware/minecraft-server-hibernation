package progmgr

import (
	"testing"
	"time"

	"github.com/shirou/gopsutil/process"
)

func Test_Upd(t *testing.T) {
	var test pStatsByPid = make(pStatsByPid)
	test.Upd(0, 1212, time.Duration(time.Second))
	test.Upd(1, 1212, time.Duration(time.Second))
	test.Upd(2, 1212, time.Duration(time.Second))

	var expected pStatsByPid = map[int32]*pStats{
		0: {1212, time.Duration(time.Second)},
		1: {1212, time.Duration(time.Second)},
		2: {1212, time.Duration(time.Second)},
	}

	if test == nil || expected == nil {
		t.Errorf("test or expected are nil")
	} else if len(test) != len(expected) {
		t.Errorf("test and expected have different lenght")
	} else {
		for pid := range test {
			if test[pid].cpuTotalLast != expected[pid].cpuTotalLast || test[pid].lifeTimeLast != expected[pid].lifeTimeLast {
				t.Errorf("test and expected have different values")
			}
		}
	}
}

func Test_Clean(t *testing.T) {
	var test pStatsByPid = make(pStatsByPid)
	test.Upd(0, 1212, time.Duration(time.Second))
	test.Upd(1, 1212, time.Duration(time.Second))
	test.Upd(2, 1212, time.Duration(time.Second))

	treeP := []*process.Process{
		{Pid: 0},
		{Pid: 3},
		{Pid: 4},
	}

	var expected pStatsByPid = make(pStatsByPid)
	expected.Upd(0, 1212, time.Duration(time.Second))

	test.Clean(treeP)

	if test == nil || expected == nil {
		t.Errorf("test or expected are nil")
	} else if len(test) != len(expected) {
		t.Errorf("test and expected have different lenght")
	} else {
		for pid := range test {
			if test[pid].cpuTotalLast != expected[pid].cpuTotalLast || test[pid].lifeTimeLast != expected[pid].lifeTimeLast {
				t.Errorf("test and expected have different values")
			}
		}
	}
}
