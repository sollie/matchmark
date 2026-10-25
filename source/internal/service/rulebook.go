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
	Ring   string
	Points int
}

type RulebookView struct {
	Courses       []RulebookCourse
	WeaponClasses map[string][]string
	Rings         []RulebookRing
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
	view.WeaponClasses = map[string][]string{}
	for _, d := range rulebook.Disciplines() {
		view.WeaponClasses[d] = rulebook.WeaponClasses(d)
	}
	for _, ring := range rulebook.RingValues() {
		points, err := rulebook.RingPoints(ring)
		if err != nil {
			return RulebookView{}, err
		}
		view.Rings = append(view.Rings, RulebookRing{Ring: ring, Points: points})
	}
	return view, nil
}
