package progmgr

import (
	"testing"
	"time"

	"github.com/shirou/gopsutil/process"
)

func Test_upd(t *testing.T) {
	var test pStatsByPid = pStatsByPid{}
	test.upd(0, 1212, time.Duration(time.Second))
	test.upd(1, 1212, time.Duration(time.Second))
	test.upd(2, 1212, time.Duration(time.Second))

	var expected pStatsByPid = pStatsByPid{
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
	var test pStatsByPid = pStatsByPid{}
	test.upd(0, 1212, time.Duration(time.Second))
	test.upd(1, 1212, time.Duration(time.Second))
	test.upd(2, 1212, time.Duration(time.Second))

	treeP := []*process.Process{
		{Pid: 0},
		{Pid: 3},
		{Pid: 4},
	}

	var expected pStatsByPid = pStatsByPid{}
	expected.upd(0, 1212, time.Duration(time.Second))

	test.clean(treeP)

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
