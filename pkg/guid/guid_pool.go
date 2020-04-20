package guid

import (
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/Mellanox/ib-kubernetes/pkg/config"
)

type Pool interface {
	// AllocateGUID allocate given guid if in range or
	// allocate the next free guid in the range if no given guid.
	// It returns the allocated guid or error if range is full.
	AllocateGUID(string) error

	GenerateGUID() (GUID, error)

	// ReleaseGUID release the reservation of the guid.
	// It returns error if the guid is not in the range.
	ReleaseGUID(string) error
}

type guidPool struct {
	rangeStart  GUID          // first guid in range
	rangeEnd    GUID          // last guid in range
	currentGUID GUID          // last given guid
	guidPoolMap map[GUID]bool // allocated guid map and status
}

func NewPool(conf *config.GUIDPoolConfig) (Pool, error) {
	log.Info().Msgf("creating guid pool, guidRangeStart %s, guidRangeEnd %s", conf.RangeStart, conf.RangeEnd)
	rangeStart, err := ParseGUID(conf.RangeStart)
	if err != nil {
		return nil, fmt.Errorf("failed to parse guidRangeStart %v", err)
	}
	rangeEnd, err := ParseGUID(conf.RangeEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to parse guidRangeStart %v", err)
	}
	if !isValidRange(rangeStart, rangeEnd) {
		return nil, fmt.Errorf("invalid guid range. rangeStart: %v rangeEnd: %v", rangeStart, rangeEnd)
	}

	return &guidPool{
		rangeStart:  rangeStart,
		rangeEnd:    rangeEnd,
		currentGUID: rangeStart,
		guidPoolMap: map[GUID]bool{},
	}, nil
}

// GenerateGUID generates a guid from the range
func (p *guidPool) GenerateGUID() (GUID, error) {
	// this look will ensure that we check all the range
	// first iteration from current guid to last guid in the range
	// second iteration from first guid in the range to the latest one
	if guid := p.getFreeGUID(p.currentGUID, p.rangeEnd); guid != 0 {
		return guid, nil
	}

	if guid := p.getFreeGUID(p.rangeStart, p.rangeEnd); guid != 0 {
		return guid, nil
	}
	return 0, fmt.Errorf("guid pool range is full")
}

// ReleaseGUID release allocated guid
func (p *guidPool) ReleaseGUID(guid string) error {
	log.Debug().Msgf("releasing guid %s", guid)
	guidAddr, err := ParseGUID(guid)
	if err != nil {
		return err
	}

	if _, ok := p.guidPoolMap[guidAddr]; !ok {
		return fmt.Errorf("failed to release guid %s, not allocated ", guid)
	}
	delete(p.guidPoolMap, guidAddr)
	return nil
}

func (p *guidPool) AllocateGUID(guid string) error {
	log.Debug().Msgf("allocating guid %s", guid)

	guidAddr, err := ParseGUID(guid)
	if err != nil {
		return err
	}

	if guidAddr < p.rangeStart || guidAddr > p.rangeEnd {
		return fmt.Errorf("out of range guid %s, pool range %v - %v", guid, p.rangeStart, p.rangeEnd)
	}

	if _, exist := p.guidPoolMap[guidAddr]; exist {
		return fmt.Errorf("failed to allocate requested guid %s, already allocated", guid)
	}

	p.guidPoolMap[guidAddr] = true
	return nil
}

func isValidRange(rangeStart, rangeEnd GUID) bool {
	return rangeStart <= rangeEnd && rangeStart != 0 && rangeEnd != 0xFFFFFFFFFFFFFFFF
}

// getFreeGUID return free guid in given range
func (p *guidPool) getFreeGUID(start, end GUID) GUID {
	for guid := start; guid <= end; guid++ {
		if _, ok := p.guidPoolMap[guid]; !ok {
			p.currentGUID++
			return guid
		}
	}

	return 0
}
