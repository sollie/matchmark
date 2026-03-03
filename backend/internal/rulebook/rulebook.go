package rulebook

import "fmt"

type Stage struct {
	StageNumber   int
	Distance      string
	ExpectedShots int
	TimeSeconds   int
	Position      string
}

type Course struct {
	Discipline string
	TotalShots int
	Stages     []Stage
}

var weaponClasses = map[string][]string{
	"1500": {
		"Revolver 1500",
		"Pistol 1500",
		"Revolver Optical Sight 1500",
		"Pistol Optical Sight 1500",
	},
	"60": {
		"Distinguished Revolver",
		"Distinguished Pistol",
		"Open Match",
	},
	"48": {
		"Standard Revolver 4.25\"",
		"Standard Revolver 2.75\"",
		"Standard Revolver 2.75\" 5-Shot",
		"Standard Semi-Automatic Pistol 5.5\" Production",
		"Standard Semi-Automatic Pistol 5.5\"",
		"Standard Semi-Automatic Pistol 3.7\"",
	},
}

const fiveShotWeaponClass = "Standard Revolver 2.75\" 5-Shot"

var course1500 = Course{
	Discipline: "1500",
	TotalShots: 150,
	Stages: []Stage{
		{1, "7 yd", 12, 20, "Standing without support"},
		{2, "15 yd", 12, 20, "Standing without support"},
		{3, "25 yd", 18, 90, "6 kneeling + 6 L-barricade + 6 R-barricade"},
		{4, "50 yd", 24, 165, "6 sitting + 6 prone + 6 L-barricade + 6 R-barricade"},
		{5, "25 yd", 12, 35, "Standing without support"},
		{6, "25 yd", 12, 35, "Standing without support"},
		{7, "7 yd", 12, 20, "Standing without support"},
		{8, "25 yd", 18, 90, "6 kneeling + 6 L-barricade + 6 R-barricade"},
		{9, "50 yd", 24, 165, "6 sitting + 6 prone + 6 L-barricade + 6 R-barricade"},
		{10, "25 yd", 6, 12, "Standing without support"},
	},
}

var course60 = Course{
	Discipline: "60",
	TotalShots: 60,
	Stages: []Stage{
		{1, "7 yd", 12, 20, "Standing without support"},
		{2, "25 yd", 18, 90, "6 kneeling + 6 L-barricade + 6 R-barricade"},
		{3, "50 yd", 24, 165, "6 sitting + 6 prone + 6 L-barricade + 6 R-barricade"},
		{4, "25 yd", 6, 12, "Standing without support"},
	},
}

var course48 = Course{
	Discipline: "48",
	TotalShots: 48,
	Stages: []Stage{
		{1, "3 yd", 6, 8, "Standing single-handed"},
		{2, "7 yd", 12, 20, "Standing without support"},
		{3, "15 yd", 12, 20, "Standing without support"},
		{4, "25 yd", 18, 90, "6 kneeling + 6 L-barricade + 6 R-barricade"},
	},
}

var course48FiveShot = Course{
	Discipline: "48",
	TotalShots: 40,
	Stages: []Stage{
		{1, "3 yd", 5, 8, "Standing single-handed"},
		{2, "7 yd", 10, 20, "Standing without support"},
		{3, "15 yd", 10, 20, "Standing without support"},
		{4, "25 yd", 15, 90, "5 kneeling + 5 L-barricade + 5 R-barricade"},
	},
}

func ValidateWeaponClass(discipline, weaponClass string) error {
	classes, ok := weaponClasses[discipline]
	if !ok {
		return fmt.Errorf("unknown discipline %q", discipline)
	}
	for _, c := range classes {
		if c == weaponClass {
			return nil
		}
	}
	return fmt.Errorf("weapon class %q is not valid for discipline %q", weaponClass, discipline)
}

func CourseOfFire(discipline, weaponClass string) (Course, error) {
	switch discipline {
	case "1500":
		return course1500, nil
	case "60":
		return course60, nil
	case "48":
		if weaponClass == fiveShotWeaponClass {
			return course48FiveShot, nil
		}
		return course48, nil
	}
	return Course{}, fmt.Errorf("unknown discipline %q", discipline)
}

func WeaponClassesForDiscipline(discipline string) ([]string, error) {
	classes, ok := weaponClasses[discipline]
	if !ok {
		return nil, fmt.Errorf("unknown discipline %q", discipline)
	}
	out := make([]string, len(classes))
	copy(out, classes)
	return out, nil
}

func RingPoints(ring string) (int, error) {
	switch ring {
	case "X", "10":
		return 10, nil
	case "9":
		return 9, nil
	case "8":
		return 8, nil
	case "7":
		return 7, nil
	case "0", "M":
		return 0, nil
	}
	return 0, fmt.Errorf("invalid ring value %q", ring)
}

func IsXCount(ring string) bool {
	return ring == "X"
}
