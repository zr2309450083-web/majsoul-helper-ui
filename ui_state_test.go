package main

import (
	"encoding/json"
	"testing"

	"github.com/EndlessCheng/mahjong-helper/util"
)

func TestUITiles34ToDTOs(t *testing.T) {
	tiles := make([]int, 34)
	tiles[0] = 2  // 1m x2
	tiles[4] = 1  // 5m (red)
	tiles[13] = 1 // 5p (not red)

	out := tiles34ToDTOs(tiles, []int{1, 0, 0})
	if len(out) != 4 {
		t.Fatalf("len = %d, want 4", len(out))
	}
	if out[0].Name != "1万" || out[1].Name != "1万" {
		t.Fatalf("unexpected man tiles: %+v", out[:2])
	}
	if out[2].Name != "5万" || !out[2].Red {
		t.Fatalf("red five not marked: %+v", out[2])
	}
	if out[3].Name != "5饼" || out[3].Red {
		t.Fatalf("non-red five marked: %+v", out[3])
	}
}

func TestUIWaitsToDTOs(t *testing.T) {
	out, total := waitsToDTOs(util.Waits{0: 1, 13: 2, 4: 0})
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2", len(out))
	}
	if out[0].Tile.Tile != 0 || out[1].Tile.Tile != 13 {
		t.Fatalf("waits not sorted: %+v", out)
	}
	if out[1].Left != 2 {
		t.Fatalf("left = %d, want 2", out[1].Left)
	}
}

func TestUIHistoryUpsert(t *testing.T) {
	var history []uiHistoryDTO
	attack := uiRecoDTO{Tile: tileDTOPtr(4), Risk: 12, RiskKnown: true}
	upsertHistory(&history, historyEntry(3, "draw", attack, uiRecoDTO{}))
	if len(history) != 1 || history[0].AIAttack == nil || history[0].AIAttack.Name != "5万" {
		t.Fatalf("attack entry not created: %+v", history)
	}

	discard := tileDTO(13)
	upsertHistory(&history, uiHistoryDTO{Turn: 3, SelfDiscard: &discard, SelfRisk: 2})
	if len(history) != 1 {
		t.Fatalf("same turn produced %d entries", len(history))
	}
	if history[0].SelfDiscard == nil || history[0].SelfDiscard.Name != "5饼" {
		t.Fatalf("self discard not merged: %+v", history[0])
	}
	if history[0].AIAttack == nil {
		t.Fatalf("attack was lost during merge: %+v", history[0])
	}
}

func TestUIResetState(t *testing.T) {
	p := &uiPublisher{state: uiStateDTO{Status: "ready", Hand: []uiTileDTO{tileDTO(0)}}}
	p.setEnabled()
	p.Reset()

	var got uiStateDTO
	if err := json.Unmarshal(p.currentJSON(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != "waiting" {
		t.Fatalf("status = %q, want waiting", got.Status)
	}
	if len(got.Hand) != 0 {
		t.Fatalf("hand not cleared: %+v", got.Hand)
	}
}
