// Copyright 2023 PingCAP, Inc. Licensed under Apache-2.0.

package input

import (
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/joomcode/errorx"

	regionpkg "github.com/pingcap/tidb-dashboard/pkg/keyvisual/region"
	"github.com/pingcap/tidb-dashboard/pkg/pd"
	"github.com/pingcap/tidb-dashboard/util/distro"
)

var (
	ErrNS          = errorx.NewNamespace("error.keyvisual")
	ErrNSInput     = ErrNS.NewSubNamespace("input")
	ErrInvalidData = ErrNSInput.NewType("invalid_data")
)

// RegionInfo records detail region info for api usage.
type RegionInfo struct {
	ID              uint64 `json:"id"`
	StartKey        string `json:"start_key"`
	EndKey          string `json:"end_key"`
	WrittenBytes    uint64 `json:"written_bytes"`
	ReadBytes       uint64 `json:"read_bytes"`
	WrittenKeys     uint64 `json:"written_keys"`
	ReadKeys        uint64 `json:"read_keys"`
	ApproximateSize int64  `json:"approximate_size"`
	ApproximateKeys int64  `json:"approximate_keys"`
}

// RegionsInfo contains some regions with the detailed region info.
type RegionsInfo struct {
	Count   int           `json:"count"`
	Regions []*RegionInfo `json:"regions"`
}

func (rs *RegionsInfo) Len() int {
	return rs.Count
}

func (rs *RegionsInfo) GetKeys() []string {
	keys := make([]string, rs.Count+1)
	keys[0] = rs.Regions[0].StartKey
	endKeys := keys[1:]
	for i, region := range rs.Regions {
		endKeys[i] = region.EndKey
	}
	return keys
}

func (rs *RegionsInfo) GetValues(tag regionpkg.StatTag) []uint64 {
	values := make([]uint64, rs.Count)
	switch tag {
	case regionpkg.WrittenBytes:
		for i, region := range rs.Regions {
			values[i] = region.WrittenBytes
		}
	case regionpkg.ReadBytes:
		for i, region := range rs.Regions {
			values[i] = region.ReadBytes
		}
	case regionpkg.WrittenKeys:
		for i, region := range rs.Regions {
			values[i] = region.WrittenKeys
		}
	case regionpkg.ReadKeys:
		for i, region := range rs.Regions {
			values[i] = region.ReadKeys
		}
	case regionpkg.Integration:
		for i, region := range rs.Regions {
			values[i] = region.WrittenBytes + region.ReadBytes
		}
	default:
		panic("unreachable")
	}
	return values
}

func read(data []byte) (*RegionsInfo, error) {
	regions := &RegionsInfo{}
	if err := json.Unmarshal(data, regions); err != nil {
		return nil, ErrInvalidData.Wrap(err, "%s regions API unmarshal failed", distro.R().PD)
	}

	for _, region := range regions.Regions {
		startBytes, err := hex.DecodeString(region.StartKey)
		if err != nil {
			return nil, ErrInvalidData.Wrap(err, "%s regions API unmarshal failed", distro.R().PD)
		}
		region.StartKey = regionpkg.String(startBytes)
		endBytes, err := hex.DecodeString(region.EndKey)
		if err != nil {
			return nil, ErrInvalidData.Wrap(err, "%s regions API unmarshal failed", distro.R().PD)
		}
		region.EndKey = regionpkg.String(endBytes)
	}

	sort.Slice(regions.Regions, func(i, j int) bool {
		return regions.Regions[i].StartKey < regions.Regions[j].StartKey
	})

	return regions, nil
}

func NewAPIPeriodicGetter(pdClient *pd.Client) regionpkg.RegionsInfoGenerator {
	return func() (regionpkg.RegionsInfo, error) {
		data, err := pdClient.SendGetRequest("/regions")
		if err != nil {
			return nil, err
		}
		return read(data)
	}
}
