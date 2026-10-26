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
	Label      string
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
	Label:      "150 shots (WA1500)",
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
	Label:      "60 shots",
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
	Label:      "48 shots",
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
	Label:      "40 shots (5-shot revolver)",
	Discipline: "48",
	TotalShots: 40,
	Stages: []Stage{
		{1, "3 yd", 5, 8, "Standing single-handed"},
		{2, "7 yd", 10, 20, "Standing without support"},
		{3, "15 yd", 10, 20, "Standing without support"},
		{4, "25 yd", 15, 90, "5 kneeling + 5 L-barricade + 5 R-barricade"},
	},
}

var courseKeys = []string{"150", "60", "48", "40"}

var disciplines = []string{"1500", "60", "48"}

var categories = []string{"Revolver", "Pistol", "Revolver OS", "Pistol OS", "Rifle"}

var courseCategories = map[string][]string{
	"150": {"Revolver", "Pistol", "Revolver OS", "Pistol OS"},
	"60":  {"Revolver", "Pistol", "Revolver OS", "Pistol OS"},
	"48":  {"Revolver", "Pistol"},
	"40":  {"Revolver"},
}

const (
	TenRingPoints   = 10
	NineRingPoints  = 9
	EightRingPoints = 8
	SevenRingPoints = 7
)

var ringValues = []string{"X", "10", "9", "8", "7", "0", "M"}

func CourseKeys() []string {
	return append([]string(nil), courseKeys...)
}

func Disciplines() []string {
	return append([]string(nil), disciplines...)
}

func Categories() []string {
	return append([]string(nil), categories...)
}

func CategoriesForCourse(key string) ([]string, error) {
	allowed, ok := courseCategories[key]
	if !ok {
		return nil, fmt.Errorf("unknown course %q", key)
	}
	return append([]string(nil), allowed...), nil
}

func ValidateCategory(courseKey, category string) error {
	allowed, err := CategoriesForCourse(courseKey)
	if err != nil {
		return err
	}
	for _, c := range allowed {
		if c == category {
			return nil
		}
	}
	return fmt.Errorf("category %q is not allowed for course %q", category, courseKey)
}

func WeaponClasses(discipline string) []string {
	return append([]string(nil), weaponClasses[discipline]...)
}

func RingValues() []string {
	return append([]string(nil), ringValues...)
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

func RingPoints(ring string) (int, error) {
	switch ring {
	case "X", "10":
		return TenRingPoints, nil
	case "9":
		return NineRingPoints, nil
	case "8":
		return EightRingPoints, nil
	case "7":
		return SevenRingPoints, nil
	case "0", "M":
		return 0, nil
	}
	return 0, fmt.Errorf("invalid ring value %q", ring)
}

func CourseByKey(key string) (Course, error) {
	switch key {
	case "150":
		return CourseOfFire("1500", "")
	case "60":
		return CourseOfFire("60", "")
	case "48":
		return CourseOfFire("48", "")
	case "40":
		return CourseOfFire("48", fiveShotWeaponClass)
	}
	return Course{}, fmt.Errorf("unknown course %q", key)
}
