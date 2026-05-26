package logic

import (
	"testing"

	"final_homework/logic-grpc-service/internal/model"
)

func TestRankSkillMentionsCountsJobRequirements(t *testing.T) {
	jobs := []model.Job{
		{Requirements: "熟悉 Go、MySQL，了解 React。"},
		{Requirements: "要求 Go/Golang 开发经验，熟悉 MySQL 和 Redis。"},
		{Requirements: "熟悉 Java，了解 mysql 调优。"},
	}

	got := rankSkillMentions(jobs)

	if len(got) < 3 {
		t.Fatalf("expected at least 3 counted skills, got %d", len(got))
	}
	counts := map[string]int{}
	for _, skill := range got {
		counts[skill.Name] = skill.Count
	}
	if counts["MySQL"] != 3 {
		t.Fatalf("expected MySQL to have 3 mentions, got %d in %+v", counts["MySQL"], got)
	}
	if counts["Go"] != 3 {
		t.Fatalf("expected Go to have 3 mentions, got %d in %+v", counts["Go"], got)
	}
}

func TestHighestEducationInApplications(t *testing.T) {
	apps := []model.ApplicationView{
		{CandidateName: "张三", JobTitle: "后端工程师", Education: "本科"},
		{CandidateName: "李四", JobTitle: "算法工程师", Education: "研究生"},
		{CandidateName: "王五", JobTitle: "前端工程师", Education: "专科"},
	}

	level, people := highestEducationInApplications(apps)

	if level != "研究生" {
		t.Fatalf("expected highest education 研究生, got %q", level)
	}
	if len(people) != 1 || people[0].CandidateName != "李四" {
		t.Fatalf("expected 李四 as highest education candidate, got %+v", people)
	}
}
