package model

type User struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	Role         string `json:"role"`
	CreatedAt    string `json:"created_at"`
}

type RefreshToken struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	TokenHash string `json:"-"`
	FamilyID  string `json:"-"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
}

type Competition struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	Date             string `json:"date"`
	LanesTotal       int    `json:"lanes_total"`
	Status           string `json:"status"`
	SpectatorEnabled bool   `json:"spectator_enabled"`
	CreatedAt        string `json:"created_at"`
}

type Shooter struct {
	ID            int64   `json:"id"`
	CompetitionID int64   `json:"competition_id"`
	Name          string  `json:"name"`
	WAID          string  `json:"wa_id"`
	Club          *string `json:"club"`
	LaneNumber    int     `json:"lane_number"`
	Discipline    string  `json:"discipline"`
	WeaponClass   string  `json:"weapon_class"`
}

type Stage struct {
	ID            int64 `json:"id"`
	ShooterID     int64 `json:"shooter_id"`
	StageNumber   int   `json:"stage_number"`
	ExpectedShots int   `json:"expected_shots"`
	FullyLocked   bool  `json:"fully_locked"`
}

type Hit struct {
	ID        int64  `json:"id"`
	StageID   int64  `json:"stage_id"`
	Ring      string `json:"ring"`
	Points    int    `json:"points"`
	CreatedAt string `json:"created_at"`
}

type StageLock struct {
	ID         int64   `json:"id"`
	StageID    int64   `json:"stage_id"`
	UserID     int64   `json:"user_id"`
	Locked     bool    `json:"locked"`
	LockedTime *string `json:"locked_time"`
}

type ScorerAssignment struct {
	ID            int64  `json:"id"`
	CompetitionID int64  `json:"competition_id"`
	ShooterID     int64  `json:"shooter_id"`
	UserID        int64  `json:"user_id"`
	Username      string `json:"username"`
	ShooterName   string `json:"shooter_name"`
	LaneNumber    int    `json:"lane_number"`
	AssignedAt    string `json:"assigned_at"`
}
