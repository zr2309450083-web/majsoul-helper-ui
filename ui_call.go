package main

import (
	"strings"

	"github.com/EndlessCheng/mahjong-helper/util"
	"github.com/EndlessCheng/mahjong-helper/util/model"
)

func firstYakuCandidate(results14, incShantenResults14 util.Hand14AnalysisResultList, table riskTable) *uiCandidateDTO {
	for i, result := range results14 {
		if result == nil || result.Result13 == nil || len(result.Result13.YakuTypes) == 0 {
			continue
		}
		candidate := candidateToDTO(result, table, i+1, false)
		return &candidate
	}
	for i, result := range incShantenResults14 {
		if result == nil || result.Result13 == nil || len(result.Result13.YakuTypes) == 0 {
			continue
		}
		candidate := candidateToDTO(result, table, i+1, true)
		return &candidate
	}
	return nil
}

func inferCallKind(candidates []uiCandidateDTO, calledTile int) string {
	if len(candidates) == 0 {
		return "鸣牌"
	}
	open := candidates[0].OpenTiles
	if len(open) >= 3 {
		return "杠"
	}
	if len(open) >= 2 && open[0].Tile == open[1].Tile {
		return "碰"
	}
	if len(open) >= 2 {
		return "吃"
	}
	return "鸣牌"
}

func ensureCandidate(list []uiCandidateDTO, best *uiCandidateDTO) []uiCandidateDTO {
	if best == nil {
		return list
	}
	for _, candidate := range list {
		if candidate.Tile.Tile == best.Tile.Tile &&
			candidate.MeldText == best.MeldText &&
			candidate.Shanten == best.Shanten {
			return list
		}
	}
	return append([]uiCandidateDTO{*best}, list...)
}

// decideCall applies a deliberately conservative riichi rule set:
//   - an open hand without a yaku can never win, so such calls are rejected;
//   - a closed hand prefers riichi unless the call is meaningfully faster
//     (or secures a yaku when the hand has none);
//   - a call that creates furiten is rejected.
//
// The result is a transparent recommendation, not a full push/fold solver.
func decideCall(playerInfo *model.PlayerInfo, calledTile int, pass *util.Hand13AnalysisResult, results14, incShantenResults14 util.Hand14AnalysisResultList, table riskTable) uiCallDTO {
	call := uiCallDTO{Available: true, Tile: tileDTO(calledTile)}
	if playerInfo == nil || pass == nil {
		call.Kind = "鸣牌"
		call.Verdict = "建议跳过"
		call.Reason = "无法分析当前手牌"
		return call
	}

	passClosed := !playerInfo.IsNaki()
	passYaku := yakuNameList(pass.YakuTypes)
	call.Pass = uiPassDTO{
		Shanten:     pass.Shanten,
		ShantenText: util.NumberToChineseShanten(pass.Shanten),
		WaitsCount:  pass.Waits.AllCount(),
		AgariRate:   pass.AvgAgariRate,
		Value:       valueOf(pass, passClosed),
		Yaku:        passYaku,
		IsClosed:    passClosed,
	}
	call.Candidates = buildCandidates(results14, incShantenResults14, table, 8)

	best := firstYakuCandidate(results14, incShantenResults14, table)
	if best == nil {
		call.Kind = inferCallKind(call.Candidates, calledTile)
		call.ShouldCall = false
		call.Verdict = "建议跳过"
		call.Reason = "无役，鸣牌后不能和牌"
		return call
	}

	call.Best = best
	call.HasYaku = true
	call.Yaku = best.Yaku
	call.Kind = inferCallKind([]uiCandidateDTO{*best}, calledTile)
	call.Furiten = best.FuritenRate >= 1
	call.Candidates = ensureCandidate(call.Candidates, best)

	callShanten := best.Shanten
	passShanten := pass.Shanten
	callSpeed := candidateSpeed(best)
	passSpeed := speedOf(pass)
	callValue := best.DamaPoint
	passValue := call.Pass.Value
	yakuStr := strings.Join(best.Yaku, "、")
	if yakuStr == "" {
		yakuStr = "役种"
	}

	switch {
	case passShanten < 0:
		call.ShouldCall = false
		call.Reason = "已经和牌"
	case call.Furiten:
		call.ShouldCall = false
		call.Reason = "鸣牌后会振听"
	case callShanten < passShanten:
		call.ShouldCall = true
		call.Reason = "有役（" + yakuStr + "）且向听前进"
	case callShanten == 0 && passShanten == 0:
		if callValue > passValue*1.5 || best.WaitsCount > pass.Waits.AllCount()+3 {
			call.ShouldCall = true
			call.Reason = "有役（" + yakuStr + "）且听牌，打点/进张明显更好"
		} else {
			call.ShouldCall = false
			call.Reason = "已听牌，保持门清立直更好"
		}
	case callShanten == passShanten && passClosed:
		if len(passYaku) == 0 && callSpeed > passSpeed*1.15 {
			call.ShouldCall = true
			call.Reason = "门清无役，鸣牌获得役种（" + yakuStr + "）且更快"
		} else {
			call.ShouldCall = false
			call.Reason = "向听未前进，保持门清立直"
		}
	case callShanten == passShanten && !passClosed:
		if callSpeed > passSpeed*1.1 || best.WaitsCount > pass.Waits.AllCount() {
			call.ShouldCall = true
			call.Reason = "有役（" + yakuStr + "），进张/速度更好"
		} else {
			call.ShouldCall = false
			call.Reason = "鸣牌不会加快"
		}
	default:
		call.ShouldCall = false
		call.Reason = "鸣牌不会加快"
	}

	if call.ShouldCall {
		call.Verdict = "建议鸣牌"
	} else {
		call.Verdict = "建议跳过"
	}
	return call
}
