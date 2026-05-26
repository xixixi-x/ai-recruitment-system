package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"final_homework/logic-grpc-service/internal/ai"
	"final_homework/logic-grpc-service/internal/model"
	"final_homework/logic-grpc-service/internal/ossstore"
	"final_homework/logic-grpc-service/internal/rpc"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct {
	db  *gorm.DB
	oss *ossstore.Store
	ai  *ai.Assistant
}

func New(db *gorm.DB, oss *ossstore.Store, assistant *ai.Assistant) *Service {
	return &Service{db: db, oss: oss, ai: assistant}
}

func (s *Service) Call(ctx context.Context, req *rpc.Request) (*rpc.Response, error) {
	switch req.Operation {
	case "auth.register":
		return s.register(req)
	case "auth.login":
		return s.login(req)
	case "job.publicList":
		return s.publicJobs(req)
	case "hr.createJob":
		return s.hrCreateJob(req)
	case "hr.listJobs":
		return s.hrListJobs(req)
	case "hr.listApplications":
		return s.hrListApplications(req)
	case "hr.applicationDetail":
		return s.hrApplicationDetail(req)
	case "hr.resumeDownloadURL":
		return s.hrResumeDownloadURL(req)
	case "hr.chat":
		return s.hrChat(ctx, req)
	case "hr.chatHistory":
		return s.hrChatHistory(req)
	case "candidate.getProfile":
		return s.candidateGetProfile(req)
	case "candidate.saveProfile":
		return s.candidateSaveProfile(req)
	case "candidate.resumeSignUpload":
		return s.candidateResumeSignUpload(req)
	case "candidate.resumeConfirm":
		return s.candidateResumeConfirm(req)
	case "candidate.applyJob":
		return s.candidateApplyJob(req)
	case "candidate.myApplications":
		return s.candidateMyApplications(req)
	default:
		return rpc.Fail(404, "unknown operation: "+req.Operation), nil
	}
}

func userID(req *rpc.Request) (uint, error) {
	idStr := req.Meta["user_id"]
	id, err := strconv.ParseUint(idStr, 10, 64)
	return uint(id), err
}

func requireRole(req *rpc.Request, role string) (uint, *rpc.Response) {
	if req.Meta["role"] != role {
		return 0, rpc.Fail(403, "permission denied")
	}
	id, err := userID(req)
	if err != nil || id == 0 {
		return 0, rpc.Fail(401, "invalid user")
	}
	return id, nil
}

type authReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (s *Service) register(req *rpc.Request) (*rpc.Response, error) {
	var body authReq
	if err := json.Unmarshal(req.Body, &body); err != nil {
		return rpc.Fail(400, "invalid body"), nil
	}
	body.Username = strings.TrimSpace(body.Username)
	if body.Username == "" || len(body.Password) < 6 {
		return rpc.Fail(400, "账号不能为空，密码至少 6 位"), nil
	}
	if body.Role != "hr" && body.Role != "candidate" {
		return rpc.Fail(400, "role must be hr or candidate"), nil
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	u := model.User{Username: body.Username, PasswordHash: string(hash), Role: body.Role}
	if err := s.db.Create(&u).Error; err != nil {
		return rpc.Fail(409, "账号已存在或创建失败"), nil
	}
	return rpc.OK(map[string]any{"id": u.ID, "username": u.Username, "role": u.Role}), nil
}

func (s *Service) login(req *rpc.Request) (*rpc.Response, error) {
	var body authReq
	if err := json.Unmarshal(req.Body, &body); err != nil {
		return rpc.Fail(400, "invalid body"), nil
	}
	var u model.User
	if err := s.db.Where("username = ? AND role = ?", body.Username, body.Role).First(&u).Error; err != nil {
		return rpc.Fail(401, "账号或密码错误"), nil
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(body.Password)) != nil {
		return rpc.Fail(401, "账号或密码错误"), nil
	}
	return rpc.OK(map[string]any{"id": u.ID, "username": u.Username, "role": u.Role}), nil
}

type paginationReq struct {
	Page, PageSize int    `json:"page"`
	Keyword        string `json:"keyword"`
}

func paginate(p, ps int) (int, int) {
	if p <= 0 {
		p = 1
	}
	if ps <= 0 || ps > 100 {
		ps = 20
	}
	return (p - 1) * ps, ps
}

func (s *Service) publicJobs(req *rpc.Request) (*rpc.Response, error) {
	var body paginationReq
	_ = json.Unmarshal(req.Body, &body)
	off, limit := paginate(body.Page, body.PageSize)
	q := s.db.Model(&model.Job{}).Where("status = ?", "open")
	if strings.TrimSpace(body.Keyword) != "" {
		kw := "%" + strings.TrimSpace(body.Keyword) + "%"
		q = q.Where("title LIKE ? OR description LIKE ? OR location LIKE ?", kw, kw, kw)
	}
	var total int64
	q.Count(&total)
	var jobs []model.Job
	q.Order("created_at desc").Offset(off).Limit(limit).Find(&jobs)
	return rpc.OK(map[string]any{"total": total, "items": jobs}), nil
}

type jobReq struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	Requirements string `json:"requirements"`
	Salary       string `json:"salary"`
	Location     string `json:"location"`
}

func (s *Service) hrCreateJob(req *rpc.Request) (*rpc.Response, error) {
	hrID, fail := requireRole(req, "hr")
	if fail != nil {
		return fail, nil
	}
	var body jobReq
	if err := json.Unmarshal(req.Body, &body); err != nil {
		return rpc.Fail(400, "invalid body"), nil
	}
	if strings.TrimSpace(body.Title) == "" {
		return rpc.Fail(400, "岗位标题不能为空"), nil
	}
	job := model.Job{HRID: hrID, Title: body.Title, Description: body.Description, Requirements: body.Requirements, Salary: body.Salary, Location: body.Location, Status: "open"}
	if err := s.db.Create(&job).Error; err != nil {
		return rpc.Fail(500, err.Error()), nil
	}
	return rpc.OK(job), nil
}

func (s *Service) hrListJobs(req *rpc.Request) (*rpc.Response, error) {
	hrID, fail := requireRole(req, "hr")
	if fail != nil {
		return fail, nil
	}
	var jobs []model.Job
	s.db.Where("hr_id = ?", hrID).Order("created_at desc").Find(&jobs)
	return rpc.OK(jobs), nil
}

func (s *Service) applicationQueryForHR(hrID uint) *gorm.DB {
	return s.db.Table("applications").
		Select(`applications.id, applications.job_id, jobs.title as job_title, applications.candidate_id,
                candidate_profiles.name as candidate_name, candidate_profiles.phone, candidate_profiles.email,
                candidate_profiles.education, candidate_profiles.school, candidate_profiles.skills,
                candidate_profiles.experience, candidate_profiles.resume_file_name, candidate_profiles.resume_object_key,
                applications.status, applications.created_at`).
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Joins("LEFT JOIN candidate_profiles ON candidate_profiles.user_id = applications.candidate_id").
		Where("jobs.hr_id = ?", hrID)
}

func (s *Service) hrListApplications(req *rpc.Request) (*rpc.Response, error) {
	hrID, fail := requireRole(req, "hr")
	if fail != nil {
		return fail, nil
	}
	var rows []model.ApplicationView
	s.applicationQueryForHR(hrID).Order("applications.created_at desc").Scan(&rows)
	return rpc.OK(rows), nil
}

func (s *Service) hrApplicationDetail(req *rpc.Request) (*rpc.Response, error) {
	hrID, fail := requireRole(req, "hr")
	if fail != nil {
		return fail, nil
	}
	var body struct {
		ID uint `json:"id"`
	}
	_ = json.Unmarshal(req.Body, &body)
	var row model.ApplicationView
	err := s.applicationQueryForHR(hrID).Where("applications.id = ?", body.ID).First(&row).Error
	if err != nil {
		return rpc.Fail(404, "投递记录不存在或无权访问"), nil
	}
	return rpc.OK(row), nil
}

func (s *Service) hrResumeDownloadURL(req *rpc.Request) (*rpc.Response, error) {
	hrID, fail := requireRole(req, "hr")
	if fail != nil {
		return fail, nil
	}
	var body struct {
		ApplicationID uint `json:"applicationId"`
	}
	_ = json.Unmarshal(req.Body, &body)
	var row model.ApplicationView
	err := s.applicationQueryForHR(hrID).Where("applications.id = ?", body.ApplicationID).First(&row).Error
	if err != nil {
		return rpc.Fail(404, "投递记录不存在或无权访问"), nil
	}
	if row.ResumeObjectKey == "" {
		return rpc.Fail(404, "候选人尚未上传合规简历"), nil
	}
	url, err := s.oss.SignedGetURL(row.ResumeObjectKey)
	if err != nil {
		return rpc.Fail(500, err.Error()), nil
	}
	return rpc.OK(map[string]any{"url": url, "filename": row.ResumeFileName}), nil
}

func (s *Service) candidateGetProfile(req *rpc.Request) (*rpc.Response, error) {
	uid, fail := requireRole(req, "candidate")
	if fail != nil {
		return fail, nil
	}
	var p model.CandidateProfile
	err := s.db.Where("user_id = ?", uid).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return rpc.OK(model.CandidateProfile{UserID: uid}), nil
	}
	if err != nil {
		return rpc.Fail(500, err.Error()), nil
	}
	return rpc.OK(p), nil
}

func (s *Service) candidateSaveProfile(req *rpc.Request) (*rpc.Response, error) {
	uid, fail := requireRole(req, "candidate")
	if fail != nil {
		return fail, nil
	}
	var body model.CandidateProfile
	if err := json.Unmarshal(req.Body, &body); err != nil {
		return rpc.Fail(400, "invalid body"), nil
	}
	body.UserID = uid
	var p model.CandidateProfile
	err := s.db.Where("user_id = ?", uid).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := s.db.Create(&body).Error; err != nil {
			return rpc.Fail(500, err.Error()), nil
		}
		return rpc.OK(body), nil
	}
	if err != nil {
		return rpc.Fail(500, err.Error()), nil
	}
	body.ID = p.ID
	body.ResumeObjectKey = p.ResumeObjectKey
	body.ResumeFileName = p.ResumeFileName
	if err := s.db.Model(&p).Updates(body).Error; err != nil {
		return rpc.Fail(500, err.Error()), nil
	}
	s.db.First(&p, p.ID)
	return rpc.OK(p), nil
}

func (s *Service) candidateResumeSignUpload(req *rpc.Request) (*rpc.Response, error) {
	uid, fail := requireRole(req, "candidate")
	if fail != nil {
		return fail, nil
	}
	var body struct {
		Filename    string `json:"filename"`
		ContentType string `json:"contentType"`
		Size        int64  `json:"size"`
	}
	_ = json.Unmarshal(req.Body, &body)
	if err := ossstore.ValidateResumeFile(body.Filename); err != nil {
		return rpc.Fail(400, err.Error()), nil
	}
	if body.Size > 10*1024*1024 {
		return rpc.Fail(400, "简历文件不能超过 10MB"), nil
	}
	key := ossstore.BuildObjectKey(uid, body.Filename)
	contentType := ossstore.ResumeContentType(body.Filename)
	url, err := s.oss.SignedPutURL(key, contentType)
	if err != nil {
		return rpc.Fail(500, err.Error()), nil
	}
	return rpc.OK(map[string]any{"uploadUrl": url, "objectKey": key, "contentType": contentType}), nil
}

func (s *Service) candidateResumeConfirm(req *rpc.Request) (*rpc.Response, error) {
	uid, fail := requireRole(req, "candidate")
	if fail != nil {
		return fail, nil
	}
	var body struct {
		ObjectKey string `json:"objectKey"`
		Filename  string `json:"filename"`
	}
	_ = json.Unmarshal(req.Body, &body)
	if !strings.HasPrefix(body.ObjectKey, fmt.Sprintf("resumes/candidate_%d/", uid)) {
		return rpc.Fail(403, "objectKey 非当前用户所有"), nil
	}
	if err := ossstore.ValidateResumeFile(body.Filename); err != nil {
		return rpc.Fail(400, err.Error()), nil
	}
	var p model.CandidateProfile
	if err := s.db.Where("user_id = ?", uid).FirstOrCreate(&p, model.CandidateProfile{UserID: uid}).Error; err != nil {
		return rpc.Fail(500, err.Error()), nil
	}
	p.ResumeObjectKey = body.ObjectKey
	p.ResumeFileName = body.Filename
	if err := s.db.Save(&p).Error; err != nil {
		return rpc.Fail(500, err.Error()), nil
	}
	return rpc.OK(p), nil
}

func (s *Service) candidateApplyJob(req *rpc.Request) (*rpc.Response, error) {
	uid, fail := requireRole(req, "candidate")
	if fail != nil {
		return fail, nil
	}
	var body struct {
		JobID uint `json:"jobId"`
	}
	_ = json.Unmarshal(req.Body, &body)
	var p model.CandidateProfile
	if err := s.db.Where("user_id = ?", uid).First(&p).Error; err != nil {
		return rpc.Fail(400, "请先完善结构化个人资料"), nil
	}
	if p.Name == "" || p.Phone == "" || p.Email == "" || p.ResumeObjectKey == "" {
		return rpc.Fail(400, "投递前必须填写姓名、电话、邮箱并上传合规简历"), nil
	}
	var job model.Job
	if err := s.db.Where("id = ? AND status = ?", body.JobID, "open").First(&job).Error; err != nil {
		return rpc.Fail(404, "岗位不存在或已关闭"), nil
	}
	app := model.Application{JobID: body.JobID, CandidateID: uid, Status: "submitted"}
	if err := s.db.Create(&app).Error; err != nil {
		return rpc.Fail(409, "该岗位已投递，请勿重复投递"), nil
	}
	return rpc.OK(app), nil
}

func (s *Service) candidateMyApplications(req *rpc.Request) (*rpc.Response, error) {
	uid, fail := requireRole(req, "candidate")
	if fail != nil {
		return fail, nil
	}
	var rows []model.ApplicationView
	s.db.Table("applications").
		Select("applications.id, applications.job_id, jobs.title as job_title, applications.candidate_id, applications.status, applications.created_at").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("applications.candidate_id = ?", uid).
		Order("applications.created_at desc").
		Scan(&rows)
	return rpc.OK(rows), nil
}

func (s *Service) hrBusinessContext(hrID uint) string {
	var totalJobs int64
	var totalApps int64
	s.db.Model(&model.Job{}).Where("hr_id = ?", hrID).Count(&totalJobs)
	s.db.Table("applications").Joins("JOIN jobs ON jobs.id = applications.job_id").Where("jobs.hr_id = ?", hrID).Count(&totalApps)
	var rows []struct {
		Title string
		Count int64
	}
	s.db.Table("jobs").Select("jobs.title, count(applications.id) as count").
		Joins("LEFT JOIN applications ON applications.job_id = jobs.id").
		Where("jobs.hr_id = ?", hrID).
		Group("jobs.id").Order("count desc").Scan(&rows)
	lines := []string{fmt.Sprintf("岗位总数: %d", totalJobs), fmt.Sprintf("投递总数: %d", totalApps), "各岗位投递统计:"}
	for _, r := range rows {
		lines = append(lines, fmt.Sprintf("- %s: %d", r.Title, r.Count))
	}
	if len(rows) == 0 {
		lines = append(lines, "- 暂无岗位数据")
	}
	return strings.Join(lines, "\n")
}

func (s *Service) hrChat(ctx context.Context, req *rpc.Request) (*rpc.Response, error) {
	hrID, fail := requireRole(req, "hr")
	if fail != nil {
		return fail, nil
	}
	var body struct {
		Question string `json:"question"`
	}
	_ = json.Unmarshal(req.Body, &body)
	q := strings.TrimSpace(body.Question)
	if q == "" {
		return rpc.Fail(400, "问题不能为空"), nil
	}
	contextText := s.hrBusinessContext(hrID)
	answer, err := s.ai.Answer(ctx, q, contextText)
	if err != nil {
		return rpc.Fail(500, err.Error()), nil
	}
	s.db.Create(&model.AIChatMessage{HRID: hrID, Role: "user", Content: q})
	s.db.Create(&model.AIChatMessage{HRID: hrID, Role: "assistant", Content: answer})
	return rpc.OK(map[string]any{"answer": answer, "businessContext": contextText}), nil
}

func (s *Service) hrChatHistory(req *rpc.Request) (*rpc.Response, error) {
	hrID, fail := requireRole(req, "hr")
	if fail != nil {
		return fail, nil
	}
	var msgs []model.AIChatMessage
	s.db.Where("hr_id = ?", hrID).Order("created_at asc").Limit(100).Find(&msgs)
	return rpc.OK(msgs), nil
}
