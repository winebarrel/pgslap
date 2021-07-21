package pgslap

import (
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/winebarrel/tachymeter"
)

type recorderDataPoint struct {
	timestamp time.Time
	resTime   time.Duration
}

type RecorderReport struct {
	URL         string
	StartedAt   time.Time
	FinishedAt  time.Time
	ElapsedTime time.Duration
	TaskOpts
	DataOpts
	GOMAXPROCS  int
	QueryCount  int
	AvgQPS      float64
	MaxQPS      float64
	MinQPS      float64
	MedianQPS   float64
	ExpectedQPS int
	Response    *tachymeter.Metrics
}

type RecorderOpts struct {
	URL       string
	HInterval time.Duration
}

type Recorder struct {
	sync.Mutex
	RecorderOpts
	TaskOpts
	DataOpts
	startedAt  time.Time
	finishedAt time.Time
	channel    chan []recorderDataPoint
	dataPoints []recorderDataPoint
}

func newRecorder(recOpts *RecorderOpts, taskOpts *TaskOpts, dataOpts *DataOpts) (rec *Recorder) {
	rec = &Recorder{
		RecorderOpts: *recOpts,
		TaskOpts:     *taskOpts,
		DataOpts:     *dataOpts,
	}

	return
}

func (rec *Recorder) start(bufsize int) {
	rec.dataPoints = []recorderDataPoint{}
	ch := make(chan []recorderDataPoint, bufsize)
	rec.channel = ch

	go func() {
		for redDps := range ch {
			rec.appendDataPoints(redDps)
		}
	}()

	rec.startedAt = time.Now()
}

func (rec *Recorder) appendDataPoints(recDps []recorderDataPoint) {
	rec.Lock()
	defer rec.Unlock()
	rec.dataPoints = append(rec.dataPoints, recDps...)
}

func (rec *Recorder) close() {
	close(rec.channel)
	rec.finishedAt = time.Now()
}

func (rec *Recorder) qpsHist() []float64 {
	recDps := rec.dataPoints

	if len(recDps) == 0 {
		return []float64{}
	}

	sort.Slice(recDps, func(i, j int) bool {
		return recDps[i].timestamp.Before(recDps[j].timestamp)
	})

	minTm := recDps[0].timestamp
	hist := []int{0}

	for _, v := range recDps {
		if minTm.Add(1 * time.Second).Before(v.timestamp) {
			minTm = minTm.Add(1 * time.Second)
			hist = append(hist, 0)
		}

		hist[len(hist)-1]++
	}

	f64Hist := make([]float64, len(hist))

	for i, v := range hist {
		f64Hist[i] = float64(v)
	}

	return f64Hist
}

func (rec *Recorder) qps() (minQPS float64, maxQPS float64, medianQPS float64) {
	qpsHist := rec.qpsHist()

	if len(qpsHist) == 0 {
		return
	}

	sort.Slice(qpsHist, func(i, j int) bool {
		return qpsHist[i] < qpsHist[j]
	})

	minQPS = qpsHist[0]
	maxQPS = qpsHist[len(qpsHist)-1]

	median := len(qpsHist) / 2
	medianNext := median + 1

	if len(qpsHist) == 1 {
		medianQPS = qpsHist[0]
	} else if len(qpsHist) == 2 {
		medianQPS = (qpsHist[0] + qpsHist[1]) / 2
	} else if len(qpsHist)%2 == 0 {
		medianQPS = (qpsHist[median] + qpsHist[medianNext]) / 2
	} else {
		medianQPS = qpsHist[medianNext]
	}

	return
}

func (rec *Recorder) add(recDps []recorderDataPoint) {
	rec.channel <- recDps
}

func (rec *Recorder) Report() (rr *RecorderReport) {
	nanoElapsed := rec.finishedAt.Sub(rec.startedAt)
	queryCnt := rec.Count()

	rr = &RecorderReport{
		URL:         rec.URL,
		StartedAt:   rec.startedAt,
		FinishedAt:  rec.finishedAt,
		ElapsedTime: nanoElapsed / time.Second,
		TaskOpts:    rec.TaskOpts,
		DataOpts:    rec.DataOpts,
		GOMAXPROCS:  runtime.GOMAXPROCS(0),
		QueryCount:  queryCnt,
		AvgQPS:      float64(queryCnt) * float64(time.Second) / float64(nanoElapsed),
		ExpectedQPS: rec.NAgents * rec.Rate,
	}

	t := tachymeter.New(&tachymeter.Config{
		Size:      len(rec.dataPoints),
		HBins:     10,
		HInterval: rec.HInterval,
	})

	for _, v := range rec.dataPoints {
		t.AddTime(v.resTime)
	}

	rr.Response = t.Calc()
	rr.MinQPS, rr.MaxQPS, rr.MedianQPS = rec.qps()

	return
}

func (rec *Recorder) Count() int {
	rec.Lock()
	defer rec.Unlock()
	return len(rec.dataPoints)
}
