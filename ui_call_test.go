package main

import (
	"strings"
	"testing"

	"github.com/EndlessCheng/mahjong-helper/util"
	"github.com/EndlessCheng/mahjong-helper/util/model"
)

func testPlayer(open bool) *model.PlayerInfo {
	player := &model.PlayerInfo{
		HandTiles34: make([]int, 34),
		LeftTiles34: make([]int, 34),
	}
	if open {
		player.Melds = []model.Meld{{
			MeldType: model.MeldTypePon,
			Tiles:    []int{0, 0, 0},
		}}
	}
	return player
}

func testResult13(shanten int, waits util.Waits, yaku map[int]struct{}, mixed, agari, dama, riichi float64) *util.Hand13AnalysisResult {
	return &util.Hand13AnalysisResult{
		Shanten:         shanten,
		Waits:           waits,
		YakuTypes:       yaku,
		MixedWaitsScore: mixed,
		AvgAgariRate:    agari,
		DamaPoint:       dama,
		RiichiPoint:     riichi,
	}
}

func testResult14(discard int, r13 *util.Hand13AnalysisResult, open ...int) *util.Hand14AnalysisResult {
	return &util.Hand14AnalysisResult{
		DiscardTile: discard,
		Result13:    r13,
		OpenTiles:   open,
	}
}

func TestDecideCallNoYaku(t *testing.T) {
	pass := testResult13(1, util.Waits{0: 4}, nil, 20, 0, 0, 0)
	call := testResult14(5, testResult13(0, util.Waits{1: 4}, nil, 0, 50, 0, 0), 0, 0)

	got := decideCall(testPlayer(false), 0, pass, util.Hand14AnalysisResultList{call}, nil, nil)
	if got.ShouldCall {
		t.Fatalf("no-yaku call must not be recommended: %+v", got)
	}
	if !strings.Contains(got.Reason, "无役") {
		t.Fatalf("reason = %q, want no-yaku explanation", got.Reason)
	}
}

func TestDecideCallYakuAdvancesShanten(t *testing.T) {
	yaku := map[int]struct{}{util.YakuTanyao: struct{}{}}
	pass := testResult13(1, util.Waits{0: 4}, nil, 20, 0, 0, 0)
	call := testResult14(5, testResult13(0, util.Waits{1: 4}, yaku, 0, 50, 2000, 0), 0, 0)

	got := decideCall(testPlayer(false), 0, pass, util.Hand14AnalysisResultList{call}, nil, nil)
	if !got.ShouldCall {
		t.Fatalf("yaku call that advances shanten should be recommended: %+v", got)
	}
	if !strings.Contains(got.Reason, "向听前进") {
		t.Fatalf("reason = %q", got.Reason)
	}
}

func TestDecideCallClosedTenpaiKeepsRiichi(t *testing.T) {
	yaku := map[int]struct{}{util.YakuTanyao: struct{}{}}
	pass := testResult13(0, util.Waits{0: 4}, yaku, 0, 50, 2000, 3000)
	call := testResult14(5, testResult13(0, util.Waits{0: 4}, yaku, 0, 50, 1000, 0), 0, 0)

	got := decideCall(testPlayer(false), 0, pass, util.Hand14AnalysisResultList{call}, nil, nil)
	if got.ShouldCall {
		t.Fatalf("closed tenpai should keep riichi: %+v", got)
	}
	if !strings.Contains(got.Reason, "门清") {
		t.Fatalf("reason = %q", got.Reason)
	}
}

func TestDecideCallOpenHandFaster(t *testing.T) {
	yaku := map[int]struct{}{util.YakuTanyao: struct{}{}}
	pass := testResult13(1, util.Waits{0: 4}, yaku, 20, 0, 1000, 0)
	call := testResult14(5, testResult13(1, util.Waits{0: 4, 1: 4}, yaku, 30, 0, 1000, 0), 0, 0)

	got := decideCall(testPlayer(true), 0, pass, util.Hand14AnalysisResultList{call}, nil, nil)
	if !got.ShouldCall {
		t.Fatalf("open hand with a yaku and better speed should be recommended: %+v", got)
	}
}

func TestDecideCallRejectsFuriten(t *testing.T) {
	yaku := map[int]struct{}{util.YakuTanyao: struct{}{}}
	pass := testResult13(1, util.Waits{0: 4}, nil, 20, 0, 0, 0)
	r13 := testResult13(1, util.Waits{0: 4}, yaku, 30, 0, 1000, 0)
	r13.FuritenRate = 1
	call := testResult14(5, r13, 0, 0)

	got := decideCall(testPlayer(true), 0, pass, util.Hand14AnalysisResultList{call}, nil, nil)
	if got.ShouldCall {
		t.Fatalf("furiten call must be rejected: %+v", got)
	}
	if !strings.Contains(got.Reason, "振听") {
		t.Fatalf("reason = %q", got.Reason)
	}
}
