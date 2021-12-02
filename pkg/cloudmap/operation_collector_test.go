package cloudmap

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	op1 = "one"
	op2 = "two"
)

func TestOpCollector_HappyCase(t *testing.T) {
	oc := NewOperationCollector()
	oc.Add(func() (opId string, err error) { return op1, nil })
	oc.Add(func() (opId string, err error) { return op2, nil })

	result := oc.Collect()
	assert.True(t, oc.IsAllOperationsCreated())
	assert.Equal(t, 2, len(result))
	assert.Contains(t, result, op1)
	assert.Contains(t, result, op2)
}

func TestOpCollector_AllFail(t *testing.T) {
	oc := NewOperationCollector()
	oc.Add(func() (opId string, err error) { return op1, errors.New("fail one") })
	oc.Add(func() (opId string, err error) { return op2, errors.New("fail two") })

	result := oc.Collect()
	assert.False(t, oc.IsAllOperationsCreated())
	assert.Equal(t, 0, len(result))
}

func TestOpCollector_MixedSuccess(t *testing.T) {
	oc := NewOperationCollector()
	oc.Add(func() (opId string, err error) { return op1, errors.New("fail one") })
	oc.Add(func() (opId string, err error) { return op2, nil })

	result := oc.Collect()
	assert.False(t, oc.IsAllOperationsCreated())
	assert.Equal(t, []string{op2}, result)
}

func TestOpCollector_GetStartTime(t *testing.T) {
	oc1 := NewOperationCollector()
	time.Sleep(time.Second)
	oc2 := NewOperationCollector()

	assert.Equal(t, oc1.GetStartTime(), oc1.GetStartTime(), "Start time should not change")
	assert.NotEqual(t, oc1.GetStartTime(), oc2.GetStartTime(), "Start time should reflect instantiation")
	assert.Less(t, oc1.GetStartTime(), oc2.GetStartTime(),
		"Start time should increase for later instantiations")
}
