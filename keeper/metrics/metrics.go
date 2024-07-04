package metrics

import (
	"github.com/Layr-Labs/eigensdk-go/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics interface {
	metrics.Metrics
	TasksReceived()
	TasksAcceptedByAggregator()
	SetUptime(value float64)
	SetValidatorPerformance(validator string, performance float64)
	SetValidatorStake(validator string, stake float64)
	BlocksProduced()
	TransactionsProcessed()
}

type AvsAndEigenMetrics struct {
	metrics.Metrics
	tasksReceived                           prometheus.Counter
	signedTaskResponsesAcceptedByAggregator prometheus.Counter
	uptime                                  prometheus.Gauge
	validatorPerformance                    *prometheus.GaugeVec
	validatorStake                          *prometheus.GaugeVec
	blocksProduced                          prometheus.Counter
	transactionsProcessed                   prometheus.Counter
}

const keeperNamespace = "keeper"

func NewAvsAndEigenMetrics(eigenMetrics *metrics.EigenMetrics, reg prometheus.Registerer) *AvsAndEigenMetrics {
	return &AvsAndEigenMetrics{
		Metrics: eigenMetrics,
		tasksReceived: promauto.With(reg).NewCounter(
			prometheus.CounterOpts{
				Namespace: keeperNamespace,
				Name:      "tasks_received",
				Help:      "The number of tasks received by reading from the avs service manager contract",
			}),
		signedTaskResponsesAcceptedByAggregator: promauto.With(reg).NewCounter(
			prometheus.CounterOpts{
				Namespace: keeperNamespace,
				Name:      "signed_task_responses_accepted_by_aggregator",
				Help:      "The number of signed task responses accepted by the aggregator",
			}),
		uptime: promauto.With(reg).NewGauge(
			prometheus.GaugeOpts{
				Namespace: keeperNamespace,
				Name:      "uptime",
				Help:      "The uptime of the service",
			}),
		validatorPerformance: promauto.With(reg).NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: keeperNamespace,
				Name:      "validator_performance",
				Help:      "Performance metrics for validators",
			},
			[]string{"validator"},
		),
		validatorStake: promauto.With(reg).NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: keeperNamespace,
				Name:      "validator_stake",
				Help:      "Stake metrics for validators",
			},
			[]string{"validator"},
		),
		blocksProduced: promauto.With(reg).NewCounter(
			prometheus.CounterOpts{
				Namespace: keeperNamespace,
				Name:      "blocks_produced",
				Help:      "The number of blocks produced",
			}),
		transactionsProcessed: promauto.With(reg).NewCounter(
			prometheus.CounterOpts{
				Namespace: keeperNamespace,
				Name:      "transactions_processed",
				Help:      "The number of transactions processed",
			}),
	}
}

func (m *AvsAndEigenMetrics) TasksReceived() {
	m.tasksReceived.Inc()
}

func (m *AvsAndEigenMetrics) TasksAcceptedByAggregator() {
	m.signedTaskResponsesAcceptedByAggregator.Inc()
}

func (m *AvsAndEigenMetrics) SetUptime(value float64) {
	m.uptime.Set(value)
}

func (m *AvsAndEigenMetrics) SetValidatorPerformance(validator string, performance float64) {
	m.validatorPerformance.WithLabelValues(validator).Set(performance)
}

func (m *AvsAndEigenMetrics) SetValidatorStake(validator string, stake float64) {
	m.validatorStake.WithLabelValues(validator).Set(stake)
}

func (m *AvsAndEigenMetrics) BlocksProduced() {
	m.blocksProduced.Inc()
}

func (m *AvsAndEigenMetrics) TransactionsProcessed() {
	m.transactionsProcessed.Inc()
}
