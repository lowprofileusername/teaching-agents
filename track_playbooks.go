package main

import (
	"fmt"
	"strings"
)

const trackNotice = "36场专家方法的迁移建议非专家本人观点、非官方评分标准、非专家为本项目定制。"

type TrackPlay struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Mechanism   string   `json:"mechanism"`
	Source      string   `json:"source"`
	Locator     string   `json:"source_locator"`
	Dimensions  []string `json:"supports_dimensions"`
	Applies     string   `json:"applies_when"`
	Caution     string   `json:"caution"`
	Attribution string   `json:"track_attribution"`
}
type TrackPlaybook struct {
	Mindset string      `json:"mindset"`
	Status  string      `json:"status"`
	Plays   []TrackPlay `json:"plays"`
}

func trackPlaybook(track string) TrackPlaybook {
	var catalog struct {
		Tracks map[string]TrackPlaybook `json:"tracks"`
	}
	if readReferenceJSON("knowledge/track_playbooks.json", &catalog) == nil {
		if p, ok := catalog.Tracks[track]; ok {
			return p
		}
	}
	return TrackPlaybook{Status: "no_traceable_source", Plays: []TrackPlay{}}
}
func installTrackPlaybooks(add func(builtinReference) error) error {
	var catalog struct {
		Tracks map[string]TrackPlaybook `json:"tracks"`
	}
	if e := readReferenceJSON("knowledge/track_playbooks.json", &catalog); e != nil {
		return e
	}
	for track, p := range catalog.Tracks {
		if track == "人工智能" {
			continue
		}
		for _, m := range p.Plays {
			if m.ID == "" || m.Source == "" || m.Locator == "" || m.Caution == "" || len(m.Dimensions) == 0 {
				return fmt.Errorf("赛道打法缺少出处字段：%s", track)
			}
			r := builtinReference{ID: "builtin:track:" + m.ID, Kind: "track_migration", Track: track, Title: track + " · " + m.Source + " · " + m.Name, Text: referenceText(m), Authority: trackNotice + " " + m.Attribution + " 警惕：" + m.Caution, SourceFile: "knowledge/track_playbooks.json", SourceURL: repoSource("knowledge/track_playbooks.json"), SourceLocator: m.Locator, Origin: m.Source, Publisher: "36场方法库按赛道整理"}
			if e := add(r); e != nil {
				return e
			}
		}
	}
	return nil
}
func trackGenerationPrompt(track string) string {
	if track == "人工智能" {
		return ""
	}
	p := trackPlaybook(track)
	if p.Status == "no_traceable_source" || len(p.Plays) == 0 {
		return "该赛道暂无可引用打法，仅通用建议。结果明确标注：该赛道暂无可引用的专家打法，以下仅为通用设计建议。不得跨赛道改名补位。"
	}
	return "本赛道研究侧重点：" + p.Mindset + "。可选教学机制库：" + referenceText(p.Plays) + "选择一到两个适配打法在questions或evidence_needed中落实，来源使用本轮builtin:track条目。" + trackNotice + "分类为分析者判断的不得说成该讲者参赛赛道；跨赛道迁移注明原赛道。成效仍待补，不复制原课数据。"
}
func queryTrack(q string) string {
	nq := normalizedQuery(q)
	for _, track := range []string{"新工科", "新医科", "新农科", "新文科", "基础课程", "课程思政", "产教融合", "实验教学"} {
		if strings.Contains(nq, track) {
			return track
		}
	}
	return ""
}
