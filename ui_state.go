package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/EndlessCheng/mahjong-helper/util"
	"github.com/EndlessCheng/mahjong-helper/util/model"
)

const uiSchemaVersion = 1

type uiTileDTO struct {
	Tile int    `json:"tile"`
	Code string `json:"code"`
	Name string `json:"name"`
	Suit string `json:"suit"`
	Red  bool   `json:"red,omitempty"`
}

type uiMeldDTO struct {
	Type     int         `json:"type"`
	TypeText string      `json:"typeText"`
	Tiles    []uiTileDTO `json:"tiles"`
}

type uiWaitDTO struct {
	Tile uiTileDTO `json:"tile"`
	Left int       `json:"left"`
}

type uiRoundDTO struct {
	Key         string `json:"key"`
	RoundNumber int    `json:"roundNumber"`
	Honba       int    `json:"honba"`
	RoundText   string `json:"roundText"`
	HonbaText   string `json:"honbaText"`
	WindText    string `json:"windText"`
	RoundWind   string `json:"roundWind"`
	SelfWind    string `json:"selfWind"`
	Dealer      int    `json:"dealer"`
	PlayerCount int    `json:"playerCount"`
}

type uiCandidateDTO struct {
	Rank            int         `json:"rank"`
	Tile            uiTileDTO   `json:"tile"`
	Shanten         int         `json:"shanten"`
	HasYaku         bool        `json:"hasYaku"`
	Yaku            []string    `json:"yaku,omitempty"`
	IsDora          bool        `json:"isDora"`
	Risk            float64     `json:"risk"`
	RiskKnown       bool        `json:"riskKnown"`
	Waits           []uiWaitDTO `json:"waits"`
	WaitsCount      int         `json:"waitsCount"`
	ImproveCount    int         `json:"improveCount"`
	AvgImproveWaits float64     `json:"avgImproveWaits"`
	AvgNextWaits    float64     `json:"avgNextWaits"`
	AgariRate       float64     `json:"agariRate"`
	MixedWaitsScore float64     `json:"mixedWaitsScore"`
	DamaPoint       float64     `json:"damaPoint"`
	RiichiPoint     float64     `json:"riichiPoint"`
	FuritenRate     float64     `json:"furitenRate"`
	DoraCount       int         `json:"doraCount"`
	OpenTiles       []uiTileDTO `json:"openTiles,omitempty"`
	MeldText        string      `json:"meldText,omitempty"`
	IsIncShanten    bool        `json:"isIncShanten"`
}

type uiRecoDTO struct {
	Tile      *uiTileDTO `json:"tile"`
	Risk      float64    `json:"risk"`
	RiskKnown bool       `json:"riskKnown"`
	Text      string     `json:"text,omitempty"`
}

type uiHistoryDTO struct {
	Turn          int        `json:"turn"`
	OpType        string     `json:"opType"`
	MeldText      string     `json:"meldText,omitempty"`
	SelfDiscard   *uiTileDTO `json:"selfDiscard"`
	SelfRisk      float64    `json:"selfRisk"`
	IsRiichi      bool       `json:"isRiichi"`
	AIAttack      *uiTileDTO `json:"aiAttack"`
	AIAttackRisk  float64    `json:"aiAttackRisk"`
	AIDefence     *uiTileDTO `json:"aiDefence"`
	AIDefenceRisk float64    `json:"aiDefenceRisk"`
}

type uiPassDTO struct {
	Shanten     int      `json:"shanten"`
	ShantenText string   `json:"shantenText"`
	WaitsCount  int      `json:"waitsCount"`
	AgariRate   float64  `json:"agariRate"`
	Value       float64  `json:"value"`
	Yaku        []string `json:"yaku"`
	IsClosed    bool     `json:"isClosed"`
}

type uiCallDTO struct {
	Available  bool             `json:"available"`
	Kind       string           `json:"kind"`
	Tile       uiTileDTO        `json:"tile"`
	ShouldCall bool             `json:"shouldCall"`
	Verdict    string           `json:"verdict"`
	Reason     string           `json:"reason"`
	HasYaku    bool             `json:"hasYaku"`
	Yaku       []string         `json:"yaku"`
	Furiten    bool             `json:"furiten"`
	Pass       uiPassDTO        `json:"pass"`
	Best       *uiCandidateDTO  `json:"best"`
	Candidates []uiCandidateDTO `json:"candidates"`
}

type uiStateDTO struct {
	SchemaVersion  int              `json:"schemaVersion"`
	Seq            uint64           `json:"seq"`
	UpdatedAt      int64            `json:"updatedAt"`
	Status         string           `json:"status"`
	Message        string           `json:"message,omitempty"`
	Mode           string           `json:"mode"`
	Round          uiRoundDTO       `json:"round"`
	Turn           int              `json:"turn"`
	Hand           []uiTileDTO      `json:"hand"`
	Drawn          *uiTileDTO       `json:"drawn"`
	Melds          []uiMeldDTO      `json:"melds"`
	DoraIndicators []uiTileDTO      `json:"doraIndicators"`
	DoraTiles      []uiTileDTO      `json:"doraTiles"`
	Shanten        int              `json:"shanten"`
	ShantenText    string           `json:"shantenText"`
	IsRiichi       bool             `json:"isRiichi"`
	LeftDraw       int              `json:"leftDraw"`
	Candidates     []uiCandidateDTO `json:"candidates"`
	Attack         uiRecoDTO        `json:"attack"`
	Defense        uiRecoDTO        `json:"defense"`
	Call           *uiCallDTO       `json:"call"`
	History        []uiHistoryDTO   `json:"history"`
}

// uiTurnContext carries only immutable copies so publishing never touches a
// roundData that another goroutine may mutate.
type uiTurnContext struct {
	Mode           string
	Round          uiRoundDTO
	Turn           int
	DrawnTile      int
	DoraIndicators []uiTileDTO
	DoraTiles      []uiTileDTO
	DefenseTile    int
	DefenseRisk    float64
	DefenseKnown   bool
	RiskTable      riskTable
}

func tileDTO(tile int) uiTileDTO {
	if tile < 0 || tile >= len(util.Mahjong) {
		return uiTileDTO{Tile: tile, Code: "?", Name: "?"}
	}
	suit := "honor"
	switch {
	case tile < 9:
		suit = "man"
	case tile < 18:
		suit = "pin"
	case tile < 27:
		suit = "sou"
	}
	return uiTileDTO{Tile: tile, Code: util.Mahjong[tile], Name: util.MahjongZH[tile], Suit: suit}
}

func tileDTOPtr(tile int) *uiTileDTO {
	if tile < 0 || tile >= len(util.Mahjong) {
		return nil
	}
	t := tileDTO(tile)
	return &t
}

func tileIndicesToDTOs(tiles []int) []uiTileDTO {
	out := make([]uiTileDTO, 0, len(tiles))
	for _, tile := range tiles {
		out = append(out, tileDTO(tile))
	}
	return out
}

// tiles34ToDTOs expands a 34-kind count array into concrete tiles. Red fives
// are best-effort: the first five of each suit is marked while red counts last.
func tiles34ToDTOs(tiles34 []int, redCounts []int) []uiTileDTO {
	out := make([]uiTileDTO, 0, 14)
	red := make([]int, 3)
	copy(red, redCounts)
	for tile := 0; tile < len(tiles34) && tile < len(util.Mahjong); tile++ {
		for n := 0; n < tiles34[tile]; n++ {
			dto := tileDTO(tile)
			if tile%9 == 4 {
				suit := tile / 9
				if suit < len(red) && red[suit] > 0 {
					dto.Red = true
					red[suit]--
				}
			}
			out = append(out, dto)
		}
	}
	return out
}

func meldsToDTOs(melds []model.Meld) []uiMeldDTO {
	out := make([]uiMeldDTO, 0, len(melds))
	for _, meld := range melds {
		typeText := "吃"
		switch meld.MeldType {
		case model.MeldTypePon:
			typeText = "碰"
		case model.MeldTypeAnkan:
			typeText = "暗杠"
		case model.MeldTypeMinkan:
			typeText = "明杠"
		case model.MeldTypeKakan:
			typeText = "加杠"
		}
		tiles := make([]uiTileDTO, 0, len(meld.Tiles))
		redUsed := false
		for _, tile := range meld.Tiles {
			dto := tileDTO(tile)
			if meld.ContainRedFive && !redUsed && tile%9 == 4 {
				dto.Red = true
				redUsed = true
			}
			tiles = append(tiles, dto)
		}
		out = append(out, uiMeldDTO{Type: meld.MeldType, TypeText: typeText, Tiles: tiles})
	}
	return out
}

func waitsToDTOs(waits util.Waits) ([]uiWaitDTO, int) {
	if len(waits) == 0 {
		return nil, 0
	}
	tiles := make([]int, 0, len(waits))
	for tile, left := range waits {
		if left > 0 {
			tiles = append(tiles, tile)
		}
	}
	sort.Ints(tiles)
	out := make([]uiWaitDTO, 0, len(tiles))
	total := 0
	for _, tile := range tiles {
		left := waits[tile]
		total += left
		out = append(out, uiWaitDTO{Tile: tileDTO(tile), Left: left})
	}
	return out, total
}

func firstRune(s string) string {
	for _, r := range s {
		return string(r)
	}
	return s
}

func meldText(openTiles []int) string {
	if len(openTiles) == 0 {
		return ""
	}
	kind := "吃"
	if len(openTiles) >= 2 && openTiles[0] == openTiles[1] {
		kind = "碰"
	}
	if len(openTiles) >= 2 {
		return kind + firstRune(tileDTO(openTiles[0]).Name) + tileDTO(openTiles[1]).Name
	}
	return kind + tileDTO(openTiles[0]).Name
}

func yakuNameList(yakuTypes map[int]struct{}) []string {
	if len(yakuTypes) == 0 {
		return nil
	}
	names := make([]string, 0, len(yakuTypes))
	for id := range yakuTypes {
		if name, ok := util.YakuNameMap[id]; ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func valueOf(r *util.Hand13AnalysisResult, closed bool) float64 {
	if r == nil {
		return 0
	}
	v := r.DamaPoint
	if closed && r.RiichiPoint > v {
		v = r.RiichiPoint
	}
	return v
}

func speedOf(r *util.Hand13AnalysisResult) float64 {
	if r == nil || r.Shanten < 0 {
		return 0
	}
	if r.Shanten == 0 {
		return r.AvgAgariRate
	}
	return r.MixedWaitsScore
}

func candidateSpeed(c *uiCandidateDTO) float64 {
	if c == nil || c.Shanten < 0 {
		return 0
	}
	if c.Shanten == 0 {
		return c.AgariRate
	}
	return c.MixedWaitsScore
}

func candidateToDTO(result *util.Hand14AnalysisResult, table riskTable, rank int, isInc bool) uiCandidateDTO {
	r13 := result.Result13
	waits, waitsCount := waitsToDTOs(r13.Waits)
	yaku := yakuNameList(r13.YakuTypes)
	risk := 0.0
	riskKnown := table != nil
	if riskKnown {
		risk = table[result.DiscardTile]
	}
	return uiCandidateDTO{
		Rank:            rank,
		Tile:            tileDTO(result.DiscardTile),
		Shanten:         r13.Shanten,
		HasYaku:         len(yaku) > 0,
		Yaku:            yaku,
		IsDora:          result.IsDiscardDoraTile,
		Risk:            risk,
		RiskKnown:       riskKnown,
		Waits:           waits,
		WaitsCount:      waitsCount,
		ImproveCount:    len(r13.Improves),
		AvgImproveWaits: r13.AvgImproveWaitsCount,
		AvgNextWaits:    r13.AvgNextShantenWaitsCount,
		AgariRate:       r13.AvgAgariRate,
		MixedWaitsScore: r13.MixedWaitsScore,
		DamaPoint:       r13.DamaPoint,
		RiichiPoint:     r13.RiichiPoint,
		FuritenRate:     r13.FuritenRate,
		DoraCount:       r13.DoraCount,
		OpenTiles:       tileIndicesToDTOs(result.OpenTiles),
		MeldText:        meldText(result.OpenTiles),
		IsIncShanten:    isInc,
	}
}

func buildCandidates(results14, incShantenResults14 util.Hand14AnalysisResultList, table riskTable, limit int) []uiCandidateDTO {
	out := make([]uiCandidateDTO, 0, limit)
	add := func(list util.Hand14AnalysisResultList, isInc bool) {
		for _, result := range list {
			if len(out) >= limit {
				return
			}
			out = append(out, candidateToDTO(result, table, len(out)+1, isInc))
		}
	}
	add(results14, false)
	if len(out) == 0 {
		add(incShantenResults14, true)
	}
	return out
}

func candidateForTile(tile int, results14, incShantenResults14 util.Hand14AnalysisResultList, table riskTable) *uiCandidateDTO {
	for i, result := range results14 {
		if result.DiscardTile == tile {
			c := candidateToDTO(result, table, i+1, false)
			return &c
		}
	}
	for i, result := range incShantenResults14 {
		if result.DiscardTile == tile {
			c := candidateToDTO(result, table, i+1, true)
			return &c
		}
	}
	return nil
}

func recoFromCandidate(c *uiCandidateDTO) uiRecoDTO {
	if c == nil {
		return uiRecoDTO{}
	}
	t := c.Tile
	return uiRecoDTO{Tile: &t, Risk: c.Risk, RiskKnown: c.RiskKnown, Text: "切 " + t.Name}
}

func recoFromDefense(ctx *uiTurnContext) uiRecoDTO {
	if ctx == nil || !ctx.DefenseKnown {
		return uiRecoDTO{}
	}
	if ctx.DefenseTile < 0 {
		return uiRecoDTO{RiskKnown: true, Text: "无危险牌"}
	}
	t := tileDTO(ctx.DefenseTile)
	return uiRecoDTO{Tile: &t, Risk: ctx.DefenseRisk, RiskKnown: true, Text: "切 " + t.Name}
}

func upsertHistory(history *[]uiHistoryDTO, entry uiHistoryDTO) {
	for i := range *history {
		if (*history)[i].Turn != entry.Turn {
			continue
		}
		old := (*history)[i]
		if entry.SelfDiscard == nil && old.SelfDiscard != nil {
			entry.SelfDiscard = old.SelfDiscard
			entry.SelfRisk = old.SelfRisk
		}
		if entry.AIAttack == nil && old.AIAttack != nil {
			entry.AIAttack = old.AIAttack
			entry.AIAttackRisk = old.AIAttackRisk
		}
		if entry.AIDefence == nil && old.AIDefence != nil {
			entry.AIDefence = old.AIDefence
			entry.AIDefenceRisk = old.AIDefenceRisk
		}
		if entry.MeldText == "" {
			entry.MeldText = old.MeldText
		}
		entry.IsRiichi = entry.IsRiichi || old.IsRiichi
		(*history)[i] = entry
		return
	}
	*history = append(*history, entry)
	if len(*history) > 30 {
		*history = (*history)[len(*history)-30:]
	}
}

func historyEntry(turn int, opType string, attack, defense uiRecoDTO) uiHistoryDTO {
	return uiHistoryDTO{
		Turn:          turn,
		OpType:        opType,
		AIAttack:      attack.Tile,
		AIAttackRisk:  attack.Risk,
		AIDefence:     defense.Tile,
		AIDefenceRisk: defense.Risk,
	}
}

type uiPublisher struct {
	enabled int32
	mu      sync.Mutex
	state   uiStateDTO
}

func waitingState() uiStateDTO {
	return uiStateDTO{
		SchemaVersion: uiSchemaVersion,
		Status:        "waiting",
		Message:       "等待牌局数据… 请开始一局游戏",
	}
}

var uiPub = &uiPublisher{state: waitingState()}

func (p *uiPublisher) setEnabled() {
	atomic.StoreInt32(&p.enabled, 1)
}

func (p *uiPublisher) isEnabled() bool {
	return atomic.LoadInt32(&p.enabled) == 1
}

// commit must be called with p.mu held; it releases the lock before fan-out.
func (p *uiPublisher) commit() {
	p.state.SchemaVersion = uiSchemaVersion
	p.state.Seq++
	p.state.UpdatedAt = time.Now().UnixMilli()
	data, err := json.Marshal(p.state)
	p.mu.Unlock()
	if err == nil {
		uiHub.broadcast(data)
	}
}

func (p *uiPublisher) currentJSON() []byte {
	p.mu.Lock()
	defer p.mu.Unlock()
	data, _ := json.Marshal(p.state)
	return data
}

func (p *uiPublisher) Reset() {
	if !p.isEnabled() {
		return
	}
	p.mu.Lock()
	p.state = waitingState()
	p.commit()
}

func (p *uiPublisher) prepareLocked(ctx *uiTurnContext, playerInfo *model.PlayerInfo) {
	if ctx.Round.Key != "" && ctx.Round.Key != p.state.Round.Key {
		p.state.History = nil
	}
	p.state.Status = "ready"
	p.state.Message = ""
	p.state.Call = nil
	p.state.Mode = ctx.Mode
	p.state.Round = ctx.Round
	p.state.Turn = ctx.Turn
	p.state.Hand = tiles34ToDTOs(playerInfo.HandTiles34, playerInfo.NumRedFives)
	p.state.Drawn = tileDTOPtr(ctx.DrawnTile)
	p.state.Melds = meldsToDTOs(playerInfo.Melds)
	p.state.DoraIndicators = ctx.DoraIndicators
	p.state.DoraTiles = ctx.DoraTiles
	p.state.IsRiichi = playerInfo.IsRiichi
	p.state.LeftDraw = playerInfo.LeftDrawTilesCount
}

func (p *uiPublisher) Publish13(ctx *uiTurnContext, playerInfo *model.PlayerInfo, result *util.Hand13AnalysisResult) {
	if !p.isEnabled() || ctx == nil || result == nil {
		return
	}
	p.mu.Lock()
	p.prepareLocked(ctx, playerInfo)
	p.state.Shanten = result.Shanten
	p.state.ShantenText = util.NumberToChineseShanten(result.Shanten)
	p.state.Candidates = nil
	p.state.Attack = uiRecoDTO{}
	p.state.Defense = recoFromDefense(ctx)
	if ctx.Mode != "analysis" && ctx.Turn > 0 {
		upsertHistory(&p.state.History, historyEntry(ctx.Turn, "draw", p.state.Attack, p.state.Defense))
	}
	p.commit()
}

func (p *uiPublisher) Publish14(ctx *uiTurnContext, playerInfo *model.PlayerInfo, shanten int, results14, incShantenResults14 util.Hand14AnalysisResultList, attackTile int) {
	if !p.isEnabled() || ctx == nil {
		return
	}
	p.mu.Lock()
	p.prepareLocked(ctx, playerInfo)
	p.state.Shanten = shanten
	p.state.ShantenText = util.NumberToChineseShanten(shanten)
	p.state.Candidates = buildCandidates(results14, incShantenResults14, ctx.RiskTable, 5)
	p.state.Attack = recoFromCandidate(candidateForTile(attackTile, results14, incShantenResults14, ctx.RiskTable))
	p.state.Defense = recoFromDefense(ctx)
	if ctx.Mode != "analysis" && ctx.Turn > 0 {
		upsertHistory(&p.state.History, historyEntry(ctx.Turn, "draw", p.state.Attack, p.state.Defense))
	}
	p.commit()
}

func (p *uiPublisher) PublishMeld(ctx *uiTurnContext, playerInfo *model.PlayerInfo, call *uiCallDTO) {
	if !p.isEnabled() || ctx == nil || call == nil || !call.Available {
		return
	}
	p.mu.Lock()
	p.prepareLocked(ctx, playerInfo)
	p.state.Call = call
	if call.ShouldCall && call.Best != nil {
		p.state.Shanten = call.Best.Shanten
		p.state.ShantenText = util.NumberToChineseShanten(call.Best.Shanten)
		p.state.Attack = recoFromCandidate(call.Best)
		p.state.Candidates = nil
	} else {
		p.state.Shanten = call.Pass.Shanten
		p.state.ShantenText = call.Pass.ShantenText
		p.state.Attack = uiRecoDTO{}
		p.state.Candidates = nil
	}
	p.state.Defense = recoFromDefense(ctx)
	if ctx.Mode != "analysis" && ctx.Turn > 0 {
		entry := historyEntry(ctx.Turn, "call", p.state.Attack, p.state.Defense)
		entry.MeldText = call.Kind + " " + call.Tile.Name + " · " + call.Verdict
		upsertHistory(&p.state.History, entry)
	}
	p.commit()
}

func (p *uiPublisher) ResetCall() {
	if !p.isEnabled() {
		return
	}
	p.mu.Lock()
	if p.state.Call == nil {
		p.mu.Unlock()
		return
	}
	p.state.Call = nil
	p.commit()
}

func (p *uiPublisher) OnSelfDiscard(tile int, risk float64, isRiichi bool) {
	if !p.isEnabled() || tile < 0 || tile >= len(util.Mahjong) {
		return
	}
	p.mu.Lock()
	t := tileDTO(tile)
	found := false
	for i := len(p.state.History) - 1; i >= 0; i-- {
		if p.state.History[i].Turn != p.state.Turn {
			continue
		}
		p.state.History[i].SelfDiscard = &t
		p.state.History[i].SelfRisk = risk
		if isRiichi {
			p.state.History[i].IsRiichi = true
			p.state.IsRiichi = true
		}
		found = true
		break
	}
	if !found {
		entry := uiHistoryDTO{Turn: p.state.Turn, OpType: "discard", SelfDiscard: &t, SelfRisk: risk, IsRiichi: isRiichi}
		upsertHistory(&p.state.History, entry)
	}
	p.commit()
}

func (p *uiPublisher) OnRoundEnd() {
	if !p.isEnabled() {
		return
	}
	p.mu.Lock()
	p.state.Status = "waiting"
	p.state.Message = "本局结束，等待下一局…"
	p.commit()
}

func roundWindText(tile int) string {
	if tile < 0 || tile >= len(util.MahjongZH) {
		return "?"
	}
	return util.MahjongZH[tile]
}

func newUITurnContext(d *roundData, playerInfo *model.PlayerInfo, mixedRiskTable riskTable, drawnTile int) *uiTurnContext {
	if d == nil || playerInfo == nil {
		return nil
	}
	roundWind := roundWindText(d.roundWindTile)
	selfWind := roundWindText(d.players[0].selfWindTile)
	ctx := &uiTurnContext{
		Mode: "match",
		Round: uiRoundDTO{
			Key:         fmt.Sprintf("%d-%d", d.roundNumber, d.benNumber),
			RoundNumber: d.roundNumber,
			Honba:       d.benNumber,
			RoundText:   roundWind + strconv.Itoa(d.roundNumber%4+1) + "局",
			HonbaText:   strconv.Itoa(d.benNumber) + "本场",
			WindText:    roundWind + "场",
			RoundWind:   roundWind,
			SelfWind:    selfWind,
			Dealer:      d.dealer,
			PlayerCount: d.playerNumber,
		},
		Turn:           len(d.players[0].discardTiles) + 1,
		DrawnTile:      drawnTile,
		DoraIndicators: tileIndicesToDTOs(d.doraIndicators),
		DoraTiles:      tileIndicesToDTOs(playerInfo.DoraTiles),
		DefenseTile:    -1,
	}
	if mixedRiskTable != nil {
		ctx.DefenseKnown = true
		ctx.DefenseTile = mixedRiskTable.getBestDefenceTile(playerInfo.HandTiles34)
		if ctx.DefenseTile >= 0 {
			ctx.DefenseRisk = mixedRiskTable[ctx.DefenseTile]
		}
		ctx.RiskTable = append(riskTable(nil), mixedRiskTable...)
	}
	return ctx
}

func newTestUITurnContext(playerInfo *model.PlayerInfo) *uiTurnContext {
	if playerInfo == nil {
		return nil
	}
	roundWind := roundWindText(playerInfo.RoundWindTile)
	selfWind := roundWindText(playerInfo.SelfWindTile)
	return &uiTurnContext{
		Mode: "analysis",
		Round: uiRoundDTO{
			Key:         "analysis",
			RoundText:   "分析模式",
			WindText:    roundWind + "场",
			RoundWind:   roundWind,
			SelfWind:    selfWind,
			PlayerCount: 4,
		},
		Turn:        0,
		DrawnTile:   -1,
		DoraTiles:   tileIndicesToDTOs(playerInfo.DoraTiles),
		DefenseTile: -1,
	}
}
