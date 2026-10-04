package profile

import (
	"path/filepath"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

const (
	healthFile         = "health.json"
	maxHealthPoints    = 200
	healthDedupeWindow = 10 * time.Minute
)

// HealthPoint is one snapshot after a profile problem check.
type HealthPoint struct {
	At       time.Time `json:"at"`
	Problems int       `json:"problems"`
	Warnings int       `json:"warnings"`
	Updates  int       `json:"updates"`
}

type healthFileData struct {
	Points []HealthPoint `json:"points"`
}

// ReadHealth returns health history points, oldest first.
func ReadHealth(dir string) ([]HealthPoint, error) {
	var file healthFileData
	if _, err := datadir.ReadJSON(filepath.Join(dir, healthFile), &file); err != nil {
		return nil, err
	}
	if file.Points == nil {
		return []HealthPoint{}, nil
	}
	return file.Points, nil
}

// AppendHealth appends a point, drops duplicates within healthDedupeWindow, and keeps the newest maxHealthPoints.
func AppendHealth(dir string, point HealthPoint) error {
	points, err := ReadHealth(dir)
	if err != nil {
		return err
	}
	if len(points) > 0 {
		last := points[len(points)-1]
		if point.Problems == last.Problems && point.Warnings == last.Warnings && point.Updates == last.Updates {
			if point.At.IsZero() {
				point.At = time.Now().UTC()
			}
			if last.At.IsZero() || point.At.Sub(last.At) < healthDedupeWindow {
				return nil
			}
		}
	}
	if point.At.IsZero() {
		point.At = time.Now().UTC()
	}
	points = append(points, point)
	if len(points) > maxHealthPoints {
		points = points[len(points)-maxHealthPoints:]
	}
	return datadir.WriteJSON(filepath.Join(dir, healthFile), healthFileData{Points: points})
}

// HealthHistory reads this profile's health snapshots, oldest first.
func (s *Store) HealthHistory(game, id string) ([]HealthPoint, error) {
	dir, err := s.ProfileDir(game, id)
	if err != nil {
		return nil, err
	}
	return ReadHealth(dir)
}
