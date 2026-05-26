package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
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

type skillMention struct {
	Name  string
	Count int
}

var skillAliases = []struct {
	Name    string
	Aliases []string
}{
	{Name: "MySQL", Aliases: []string{"mysql"}},
	{Name: "Go", Aliases: []string{"go", "golang"}},
	{Name: "Java", Aliases: []string{"java"}},
	{Name: "Python", Aliases: []string{"python"}},
	{Name: "JavaScript", Aliases: []string{"javascript", "js"}},
	{Name: "TypeScript", Aliases: []string{"typescript", "ts"}},
	{Name: "React", Aliases: []string{"react"}},
	{Name: "Vue", Aliases: []string{"vue"}},
	{Name: "Redis", Aliases: []string{"redis"}},
	{Name: "Docker", Aliases: []string{"docker"}},
	{Name: "Kubernetes", Aliases: []string{"kubernetes", "k8s"}},
	{Name: "Linux", Aliases: []string{"linux"}},
	{Name: "微服务", Aliases: []string{"微服务"}},
	{Name: "gRPC", Aliases: []string{"grpc"}},
	{Name: "Gin", Aliases: []string{"gin"}},
	{Name: "算法", Aliases: []string{"算法"}},
	{Name: "机器学习", Aliases: []string{"机器学习", "machine learning"}},
	{Name: "大模型", Aliases: []string{"大模型", "llm"}},
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

func aliasRanges(text, alias string) [][2]int {
	if strings.IndexFunc(alias, func(r rune) bool {
		return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
	}) == -1 {
		var ranges [][2]int
		offset := 0
		for {
			idx := strings.Index(text[offset:], alias)
			if idx == -1 {
				break
			}
			start := offset + idx
			end := start + len(alias)
			ranges = append(ranges, [2]int{start, end})
			offset = end
		}
		return ranges
	}
	pattern := regexp.MustCompile(`(^|[^a-z0-9])` + regexp.QuoteMeta(alias) + `([^a-z0-9]|$)`)
	matches := pattern.FindAllStringSubmatchIndex(text, -1)
	ranges := make([][2]int, 0, len(matches))
	for _, match := range matches {
		start := match[0]
		end := match[1]
		for start < end && ((text[start] < 'a' || text[start] > 'z') && (text[start] < '0' || text[start] > '9')) {
			start++
		}
		for end > start && ((text[end-1] < 'a' || text[end-1] > 'z') && (text[end-1] < '0' || text[end-1] > '9')) {
			end--
		}
		ranges = append(ranges, [2]int{start, end})
	}
	return ranges
}

func countAliases(text string, aliases []string) int {
	sortedAliases := append([]string(nil), aliases...)
	sort.Slice(sortedAliases, func(i, j int) bool {
		return len(sortedAliases[i]) > len(sortedAliases[j])
	})
	used := make([]bool, len(text))
	total := 0
	for _, alias := range sortedAliases {
		for _, r := range aliasRanges(text, strings.ToLower(alias)) {
			overlap := false
			for i := r[0]; i < r[1]; i++ {
				if used[i] {
					overlap = true
					break
				}
			}
			if overlap {
				continue
			}
			for i := r[0]; i < r[1]; i++ {
				used[i] = true
			}
			total++
		}
	}
	return total
}

func rankSkillMentions(jobs []model.Job) []skillMention {
	counts := make([]skillMention, 0, len(skillAliases))
	for _, skill := range skillAliases {
		total := 0
		for _, job := range jobs {
			text := strings.ToLower(job.Title + "\n" + job.Description + "\n" + job.Requirements)
			total += countAliases(text, skill.Aliases)
		}
		if total > 0 {
			counts = append(counts, skillMention{Name: skill.Name, Count: total})
		}
	}
	sort.Slice(counts, func(i, j int) bool {
		if counts[i].Count == counts[j].Count {
			return counts[i].Name < counts[j].Name
		}
		return counts[i].Count > counts[j].Count
	})
	return counts
}

func educationRank(education string) int {
	switch strings.TrimSpace(education) {
	case "博士", "博士生", "博士研究生":
		return 6
	case "硕士", "硕士生", "研究生", "硕士研究生":
		return 5
	case "本科", "学士":
		return 4
	case "专科", "大专":
		return 3
	case "高中":
		return 2
	case "初中":
		return 1
	default:
		return 0
	}
}

func highestEducationInApplications(apps []model.ApplicationView) (string, []model.ApplicationView) {
	highestRank := 0
	highest := ""
	var people []model.ApplicationView
	for _, app := range apps {
		level := strings.TrimSpace(app.Education)
		rank := educationRank(level)
		if level == "" || rank == 0 {
			continue
		}
		if rank > highestRank {
			highestRank = rank
			highest = level
			people = []model.ApplicationView{app}
			continue
		}
		if rank == highestRank {
			people = append(people, app)
		}
	}
	return highest, people
}

func truncateText(s string, max int) string {
	s = strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
	if len([]rune(s)) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max]) + "..."
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

	var jobs []model.Job
	s.db.Where("hr_id = ?", hrID).Order("created_at desc").Limit(100).Find(&jobs)

	var apps []model.ApplicationView
	s.applicationQueryForHR(hrID).Order("applications.created_at desc").Limit(200).Scan(&apps)

	lines = append(lines, "", "岗位要求技能提及频次:")
	skills := rankSkillMentions(jobs)
	if len(skills) == 0 {
		lines = append(lines, "- 暂无可识别技能关键词")
	} else {
		for i, skill := range skills {
			if i >= 10 {
				break
			}
			lines = append(lines, fmt.Sprintf("- %s: %d 次", skill.Name, skill.Count))
		}
	}

	educationCounts := map[string]int{}
	for _, app := range apps {
		if edu := strings.TrimSpace(app.Education); edu != "" {
			educationCounts[edu]++
		}
	}
	highestEducation, highestPeople := highestEducationInApplications(apps)
	lines = append(lines, "", "投递候选人学历分布:")
	if len(educationCounts) == 0 {
		lines = append(lines, "- 暂无候选人学历数据")
	} else {
		educations := make([]string, 0, len(educationCounts))
		for edu := range educationCounts {
			educations = append(educations, edu)
		}
		sort.Slice(educations, func(i, j int) bool {
			ri, rj := educationRank(educations[i]), educationRank(educations[j])
			if ri == rj {
				return educations[i] < educations[j]
			}
			return ri > rj
		})
		for _, edu := range educations {
			lines = append(lines, fmt.Sprintf("- %s: %d 人", edu, educationCounts[edu]))
		}
		lines = append(lines, fmt.Sprintf("- 当前最高学历: %s", highestEducation))
		for _, person := range highestPeople {
			name := person.CandidateName
			if name == "" {
				name = "未填写姓名"
			}
			lines = append(lines, fmt.Sprintf("  - %s，应聘 %s", name, person.JobTitle))
		}
	}

	lines = append(lines, "", "岗位明细:")
	if len(jobs) == 0 {
		lines = append(lines, "- 暂无岗位明细")
	} else {
		for _, job := range jobs {
			lines = append(lines, fmt.Sprintf("- 岗位: %s | 地点: %s | 薪资: %s | 状态: %s", job.Title, job.Location, job.Salary, job.Status))
			lines = append(lines, fmt.Sprintf("  描述: %s", truncateText(job.Description, 180)))
			lines = append(lines, fmt.Sprintf("  要求: %s", truncateText(job.Requirements, 260)))
		}
	}

	lines = append(lines, "", "投递与候选人档案明细:")
	if len(apps) == 0 {
		lines = append(lines, "- 暂无投递明细")
	} else {
		for _, app := range apps {
			name := app.CandidateName
			if name == "" {
				name = "未填写姓名"
			}
			lines = append(lines, fmt.Sprintf("- 候选人: %s | 应聘岗位: %s | 学历: %s | 学校: %s | 状态: %s | 投递时间: %s", name, app.JobTitle, app.Education, app.School, app.Status, app.CreatedAt.Format("2006-01-02 15:04")))
			lines = append(lines, fmt.Sprintf("  技能: %s", truncateText(app.Skills, 180)))
			lines = append(lines, fmt.Sprintf("  经历: %s", truncateText(app.Experience, 240)))
			lines = append(lines, fmt.Sprintf("  简历文件: %s", app.ResumeFileName))
		}
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
