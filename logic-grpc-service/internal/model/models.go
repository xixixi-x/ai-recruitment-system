package model

import "time"

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Role         string    `gorm:"index;size:20;not null" json:"role"` // hr / candidate
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Job struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	HRID         uint      `gorm:"index;not null" json:"hrId"`
	Title        string    `gorm:"size:120;not null" json:"title"`
	Description  string    `gorm:"type:text" json:"description"`
	Requirements string    `gorm:"type:text" json:"requirements"`
	Salary       string    `gorm:"size:120" json:"salary"`
	Location     string    `gorm:"size:120" json:"location"`
	Status       string    `gorm:"size:20;default:'open'" json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type CandidateProfile struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserID          uint      `gorm:"uniqueIndex;not null" json:"userId"`
	Name            string    `gorm:"size:80" json:"name"`
	Phone           string    `gorm:"size:40" json:"phone"`
	Email           string    `gorm:"size:120" json:"email"`
	Education       string    `gorm:"size:120" json:"education"`
	School          string    `gorm:"size:120" json:"school"`
	Experience      string    `gorm:"type:text" json:"experience"`
	Skills          string    `gorm:"type:text" json:"skills"`
	ResumeObjectKey string    `gorm:"size:255" json:"resumeObjectKey"`
	ResumeFileName  string    `gorm:"size:255" json:"resumeFileName"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type Application struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	JobID       uint      `gorm:"uniqueIndex:idx_job_candidate;not null" json:"jobId"`
	CandidateID uint      `gorm:"uniqueIndex:idx_job_candidate;not null" json:"candidateId"`
	Status      string    `gorm:"size:30;default:'submitted'" json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type AIChatMessage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	HRID      uint      `gorm:"index;not null" json:"hrId"`
	Role      string    `gorm:"size:20;not null" json:"role"` // user / assistant
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type ApplicationView struct {
	ID              uint      `json:"id"`
	JobID           uint      `json:"jobId"`
	JobTitle        string    `json:"jobTitle"`
	CandidateID     uint      `json:"candidateId"`
	CandidateName   string    `json:"candidateName"`
	Phone           string    `json:"phone"`
	Email           string    `json:"email"`
	Education       string    `json:"education"`
	School          string    `json:"school"`
	Skills          string    `json:"skills"`
	Experience      string    `json:"experience"`
	ResumeFileName  string    `json:"resumeFileName"`
	ResumeObjectKey string    `json:"-"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"createdAt"`
}
