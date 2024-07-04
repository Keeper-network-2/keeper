package metrics

import (
	"testing"

	"github.com/Layr-Labs/eigensdk-go/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	eigenMetrics := &metrics.EigenMetrics{} // Change this line
	m := NewAvsAndEigenMetrics(eigenMetrics, reg)

	// Test TasksReceived
	m.TasksReceived()
	if testutil.ToFloat64(m.tasksReceived) != 1 {
		t.Errorf("tasksReceived should be 1, got %f", testutil.ToFloat64(m.tasksReceived))
	}

	// Test TasksAcceptedByAggregator
	m.TasksAcceptedByAggregator()
	if testutil.ToFloat64(m.signedTaskResponsesAcceptedByAggregator) != 1 {
		t.Errorf("signedTaskResponsesAcceptedByAggregator should be 1, got %f", testutil.ToFloat64(m.signedTaskResponsesAcceptedByAggregator))
	}

	// Test SetUptime
	m.SetUptime(99.9)
	if testutil.ToFloat64(m.uptime) != 99.9 {
		t.Errorf("uptime should be 99.9, got %f", testutil.ToFloat64(m.uptime))
	}

	// Test SetValidatorPerformance
	m.SetValidatorPerformance("validator1", 95.5)
	if testutil.ToFloat64(m.validatorPerformance.WithLabelValues("validator1")) != 95.5 {
		t.Errorf("validatorPerformance should be 95.5, got %f", testutil.ToFloat64(m.validatorPerformance.WithLabelValues("validator1")))
	}

	// Test SetValidatorStake
	m.SetValidatorStake("validator1", 1000.0)
	if testutil.ToFloat64(m.validatorStake.WithLabelValues("validator1")) != 1000.0 {
		t.Errorf("validatorStake should be 1000.0, got %f", testutil.ToFloat64(m.validatorStake.WithLabelValues("validator1")))
	}

	// Test BlocksProduced
	m.BlocksProduced()
	if testutil.ToFloat64(m.blocksProduced) != 1 {
		t.Errorf("blocksProduced should be 1, got %f", testutil.ToFloat64(m.blocksProduced))
	}

	// Test TransactionsProcessed
	m.TransactionsProcessed()
	if testutil.ToFloat64(m.transactionsProcessed) != 1 {
		t.Errorf("transactionsProcessed should be 1, got %f", testutil.ToFloat64(m.transactionsProcessed))
	}
}
