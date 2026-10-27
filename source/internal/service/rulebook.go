package service

import "github.com/sollie/matchmark/internal/rulebook"

type RulebookStage struct {
	StageNumber   int
	Distance      string
	ExpectedShots int
	TimeSeconds   int
	Position      string
}

type RulebookCourse struct {
	Key        string
	Label      string
	Discipline string
	TotalShots int
	Categories []string
	Stages     []RulebookStage
}

type RulebookRing struct {
	Ring  string
	Value int
}

type RulebookView struct {
	Courses []RulebookCourse
	Rings   []RulebookRing
}

func (s *Service) Rulebook() (RulebookView, error) {
	var view RulebookView
	for _, key := range rulebook.CourseKeys() {
		c, err := rulebook.CourseByKey(key)
		if err != nil {
			return RulebookView{}, err
		}
		allowed, err := rulebook.CategoriesForCourse(key)
		if err != nil {
			return RulebookView{}, err
		}
		course := RulebookCourse{Key: key, Label: c.Label, Discipline: c.Discipline, TotalShots: c.TotalShots, Categories: allowed}
		for _, st := range c.Stages {
			course.Stages = append(course.Stages, RulebookStage(st))
		}
		view.Courses = append(view.Courses, course)
	}
	for _, ring := range rulebook.RingValues() {
		value, err := rulebook.RingValue(ring)
		if err != nil {
			return RulebookView{}, err
		}
		view.Rings = append(view.Rings, RulebookRing{Ring: ring, Value: value})
	}
	return view, nil
}
