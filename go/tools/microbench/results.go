/*
 *
 * Copyright 2021 The Vitess Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 * /
 */

package microbench

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/vitessio/arewefastyet/go/mysql"
	"github.com/vitessio/arewefastyet/go/tools/math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type (
	// MicroBenchmarkResult contains all the metrics measured by a microbenchmark.
	MicroBenchmarkResult struct {
		Ops         int
		NSPerOp     float64
		MBPerSec    float64
		BytesPerOp  float64
		AllocsPerOp float64
	}

	// BenchmarkId represents the identification of a microbenchmark.
	BenchmarkId struct {
		PkgName string
		Name    string
	}

	// MicroBenchmarkDetails refers to a single microbenchmark.
	MicroBenchmarkDetails struct {
		BenchmarkId
		GitRef string
		Result MicroBenchmarkResult
	}

	// MicroBenchmarkComparison allows comparison of two MicroBenchmarkResult
	// that share the same BenchmarkId.
	MicroBenchmarkComparison struct {
		BenchmarkId
		Current, Last MicroBenchmarkResult

		// Difference of NSPerOp in % for Current and Last.
		CurrLastDiff float64
	}

	MicroBenchmarkDetailsArray    []MicroBenchmarkDetails
	MicroBenchmarkComparisonArray []MicroBenchmarkComparison
)

// NewMicroBenchmarkDetails creates a new MicroBenchmarkDetails.
func NewMicroBenchmarkDetails(benchmarkId BenchmarkId, gitRef string, result MicroBenchmarkResult) *MicroBenchmarkDetails {
	return &MicroBenchmarkDetails{
		BenchmarkId: benchmarkId,
		GitRef:      gitRef,
		Result:      result,
	}
}

// NewBenchmarkId creates a new BenchmarkId.
func NewBenchmarkId(pkgName string, name string) *BenchmarkId {
	return &BenchmarkId{
		PkgName: pkgName,
		Name:    name,
	}
}

// NewMicroBenchmarkResult creates a new MicroBenchmarkResult.
func NewMicroBenchmarkResult(ops int, NSPerOp, MBPerSec, BytesPerOp, AllocsPerOp float64) *MicroBenchmarkResult {
	return &MicroBenchmarkResult{
		Ops:         ops,
		NSPerOp:     NSPerOp,
		MBPerSec:    MBPerSec,
		BytesPerOp:  BytesPerOp,
		AllocsPerOp: AllocsPerOp,
	}
}

// MergeMicroBenchmarkDetails merges two MicroBenchmarkDetailsArray into a single
// MicroBenchmarkComparisonArray.
func MergeMicroBenchmarkDetails(currentMbd, lastReleaseMbd MicroBenchmarkDetailsArray) (compareMbs MicroBenchmarkComparisonArray) {
	for _, details := range currentMbd {
		var compareMb MicroBenchmarkComparison
		compareMb.BenchmarkId = details.BenchmarkId
		compareMb.Current = details.Result
		compareMb.CurrLastDiff = 1.00
		for j := 0; j < len(lastReleaseMbd); j++ {
			if lastReleaseMbd[j].BenchmarkId == details.BenchmarkId {
				compareMb.Last = lastReleaseMbd[j].Result
				compareMb.CurrLastDiff =  compareMb.Last.NSPerOp / compareMb.Current.NSPerOp
				break
			}
		}
		compareMbs = append(compareMbs, compareMb)
	}
	return compareMbs
}

// ReduceSimpleMedian reduces a MicroBenchmarkDetailsArray by merging
// all MicroBenchmarkDetails with the same benchmark name into a single
// one. The results of each MicroBenchmarkDetails correspond to the median
// of the merged elements.
func (mbd MicroBenchmarkDetailsArray) ReduceSimpleMedian() (reduceMbd MicroBenchmarkDetailsArray) {
	sort.SliceStable(mbd, func(i, j int) bool {
		return mbd[i].Name < mbd[j].Name
	})
	sort.SliceStable(mbd, func(i, j int) bool {
		return mbd[i].PkgName < mbd[j].PkgName
	})
	for i := 0; i < len(mbd); {
		var j int
		var interOps []int
		var interNSPerOp []float64
		var interMBPerSec []float64
		var interBytesPerOp []float64
		var interAllocsPerOp []float64
		for j = i; j < len(mbd) && mbd[i].Name == mbd[j].Name; j++ {
			interOps = append(interOps, mbd[j].Result.Ops)
			interNSPerOp = append(interNSPerOp, mbd[j].Result.NSPerOp)
			interMBPerSec = append(interMBPerSec, mbd[j].Result.MBPerSec)
			interBytesPerOp = append(interBytesPerOp, mbd[j].Result.BytesPerOp)
			interAllocsPerOp = append(interAllocsPerOp, mbd[j].Result.AllocsPerOp)
		}

		interOpsResult := int(math.MedianInt(interOps))
		interNSPerOpResult := math.MedianFloat(interNSPerOp)
		interMBPerSecResult := math.MedianFloat(interMBPerSec)
		interBytesPerOpResult := math.MedianFloat(interBytesPerOp)
		interAllocsPerOpResult := math.MedianFloat(interAllocsPerOp)
		reduceMbd = append(reduceMbd, *NewMicroBenchmarkDetails(
			*NewBenchmarkId(mbd[i].PkgName, mbd[i].Name),
			mbd[i].GitRef,
			*NewMicroBenchmarkResult(interOpsResult, interNSPerOpResult, interMBPerSecResult, interBytesPerOpResult, interAllocsPerOpResult),
		))
		i = j
	}
	return reduceMbd
}

// GetResultsForGitRef will fetch and return a MicroBenchmarkDetailsArray
// containing all the MicroBenchmarkDetails linked to the given git commit SHA.
func GetResultsForGitRef(ref string, client *mysql.Client) (mrs MicroBenchmarkDetailsArray, err error) {
	rows, err := client.Select("select m.pkg_name, m.name, md.n, md.ns_per_op, md.bytes_per_op,"+
		" md.allocs_per_op, md.mb_per_sec FROM microbenchmark m, microbenchmark_details md where m.git_ref = ? AND "+
		"md.microbenchmark_no = m.microbenchmark_no order by m.microbenchmark_no desc", ref)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var res MicroBenchmarkDetails
		res.GitRef = ref
		err = rows.Scan(&res.PkgName, &res.Name, &res.Result.Ops, &res.Result.NSPerOp, &res.Result.BytesPerOp,
			&res.Result.AllocsPerOp, &res.Result.MBPerSec)
		if err != nil {
			return nil, err
		}
		mrs = append(mrs, res)
	}
	return mrs, nil
}

func (r MicroBenchmarkResult) OpsStr() string {
	if r.Ops == 0 {
		return ""
	}
	return humanize.Comma(int64(r.Ops))
}

func (r MicroBenchmarkResult) NSPerOpStr() string {
	if r.NSPerOp == 0 {
		return ""
	}

	return humanize.FormatFloat("#,###.#",r.NSPerOp)
}

func (r MicroBenchmarkResult) NSPerOpToDurationStr() string {
	if r.NSPerOp == 0 {
		return ""
	}

	dur, _ := time.ParseDuration(fmt.Sprintf("%fns", r.NSPerOp))
	str := dur.String()
	i := strings.IndexFunc(str, func(r rune) bool {
		return !unicode.IsNumber(r) && r != '.'
	})
	durStr := str[:i]
	durUnit := str[i:]
	durFloat, _ := strconv.ParseFloat(durStr, 64)
	return fmt.Sprintf("%.2f %s", durFloat, durUnit)
}

func (r MicroBenchmarkResult) MBPerSecStr() string {
	if r.MBPerSec == 0 {
		return ""
	}

	return humanize.Bytes(uint64(r.MBPerSec)) + "/s"
}
func (r MicroBenchmarkResult) BytesPerOpStr() string {
	if r.BytesPerOp == 0 {
		return ""
	}

	return humanize.Bytes(uint64(r.BytesPerOp)) + "/s"
}
func (r MicroBenchmarkResult) AllocsPerOpStr() string {
	if r.AllocsPerOp == 0 {
		return ""
	}

	return humanize.SI(r.AllocsPerOp, "")
}

func (r MicroBenchmarkComparison) CurrLastDiffStr() string {

	return humanize.Comma(int64(r.CurrLastDiff*100)) + "%"
}
