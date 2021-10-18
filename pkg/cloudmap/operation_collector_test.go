package cloudmap

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestOpCollector_HappyCase(t *testing.T) {
	oc := NewOperationCollector()
	oc.Add(func() (opId string, err error) { return "one", nil })
	oc.Add(func() (opId string, err error) { return "two", nil })

	result := oc.Collect()
	assert.True(t, oc.IsAllOperationsCreated())
	assert.Equal(t, 2, len(result))
	assert.Contains(t, result, "one")
	assert.Contains(t, result, "two")
}

func TestOpCollector_AllFail(t *testing.T) {
	oc := NewOperationCollector()
	oc.Add(func() (opId string, err error) { return "one", errors.New("fail one") })
	oc.Add(func() (opId string, err error) { return "two", errors.New("fail two") })

	result := oc.Collect()
	assert.False(t, oc.IsAllOperationsCreated())
	assert.Equal(t, 0, len(result))
}

func TestOpCollector_MixedSuccess(t *testing.T) {
	oc := NewOperationCollector()
	oc.Add(func() (opId string, err error) { return "one", errors.New("fail one") })
	oc.Add(func() (opId string, err error) { return "two", nil })

	result := oc.Collect()
	assert.False(t, oc.IsAllOperationsCreated())
	assert.Equal(t, []string{"two"}, result)
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
