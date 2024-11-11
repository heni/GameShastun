package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"slices"
	"strings"
)

type BoardState int
type Move [2]int
type StrategyOptions struct {
	do_optimal_selection bool
	do_optimal_skip      bool
}
type PrecalcSkip struct {
	speedup       float64
	moves_to_skip []Move
}
type TimedMove struct {
	time_to_win float64
	move        Move
}

func update_state_1(s BoardState, m Move) (BoardState, bool) {
	x := (1 << (m[0] - 1)) | (1 << (m[1] - 1))
	if int(s)&x == x {
		return BoardState(int(s) ^ x), true
	}
	return s, false
}

func update_state_2(s BoardState, m Move) (BoardState, bool) {
	x := (1 << (m[0] + m[1] - 1))
	if int(s)&x == x {
		return BoardState(int(s) ^ x), true
	}
	return s, false
}

func nbits(v int) int {
	k := 0
	for ; v > 0; k++ {
		v = v & (v - 1)
	}
	return k
}

func state_to_string(s BoardState) string {
	var builder strings.Builder
	builder.WriteString("{")
	pos := int(s)
	for p := 1; p <= 12; p++ {
		if pos&(1<<(p-1)) != 0 {
			if pos >= 1<<p {
				builder.WriteString(fmt.Sprintf("%d,", p))
			} else {
				builder.WriteString(fmt.Sprintf("%d", p))
			}
		}
	}
	builder.WriteString("}")
	return builder.String()
}

func calc_game_table(strategy StrategyOptions, skips_table *[]PrecalcSkip) []float64 {
	const N = 1 << 12

	time_to_win := make([]float64, N)

	calculation_queue := make([]int, 0)
	for i := 0; i < N; i++ {
		calculation_queue = append(calculation_queue, i)
	}
	slices.SortStableFunc(calculation_queue, func(l, r int) int {
		return nbits(l) - nbits(r)
	})

	time_to_win[0] = 0.0
	for _, pos := range calculation_queue[1:] {

		next_moves := make([]TimedMove, 0)
		skips := 0

		for a := 1; a <= 6; a++ {
			for b := 1; b <= 6; b++ {
				s1, ok1 := update_state_1(BoardState(pos), Move{a, b})
				s2, ok2 := update_state_2(BoardState(pos), Move{a, b})
				if ok1 && ok2 {
					var tm float64
					if strategy.do_optimal_selection {
						tm = min(time_to_win[int(s1)], time_to_win[int(s2)])
					} else {
						tm = .5*time_to_win[int(s1)] + .5*time_to_win[int(s2)]
					}
					next_moves = append(next_moves, TimedMove{tm, Move{a, b}})
				} else if ok1 {
					next_moves = append(next_moves, TimedMove{time_to_win[int(s1)], Move{a, b}})
				} else if ok2 {
					next_moves = append(next_moves, TimedMove{time_to_win[int(s2)], Move{a, b}})
				} else {
					skips++
				}
			}
		}

		sum_tm := 0.0
		for _, move := range next_moves {
			sum_tm += move.time_to_win
		}
		best_time := (36 + sum_tm) / (36 - float64(skips))

		if strategy.do_optimal_skip {
			if speedup_time, moves_to_skip := prune_bad_moves(next_moves, &best_time); moves_to_skip != nil {
				if skips_table != nil {
					(*skips_table)[pos] = PrecalcSkip{speedup_time, moves_to_skip}
				}
			}
		}

		time_to_win[pos] = best_time
	}

	return time_to_win
}

func prune_bad_moves(moves []TimedMove, best_time *float64) (float64, []Move) {
	slices.SortFunc(moves, func(a, b TimedMove) int {
		return int(math.Copysign(1, a.time_to_win-b.time_to_win))
	})

	skips := 36 - len(moves)
	sum_tm := 0.0
	for _, move := range moves {
		sum_tm += move.time_to_win
	}

	skip_tm, speedup_time := 0.0, 0.0
	extra_skips := 0
	moves_to_skip := make([]Move, 0)
	for i := len(moves) - 1; i > 0; i-- {
		skip_tm += moves[i].time_to_win
		extra_skips++
		cand_time := (36 + sum_tm - skip_tm) / (36 - float64(skips+extra_skips))
		if cand_time > *best_time {
			break
		}

		moves_to_skip = append(moves_to_skip, moves[i].move)
		speedup_time += *best_time - cand_time
		*best_time = cand_time
	}

	if len(moves_to_skip) > 0 {
		return speedup_time, moves_to_skip
	}

	return 0.0, nil
}

func save_game_table(game []float64, filename string) {
	f, err := os.Create(filename)
	if err != nil {
		panic("can't create output file")
	}

	buffer := bufio.NewWriter(f)
	defer func() {
		buffer.Flush()
		f.Close()
	}()

	for pos := range game {
		buffer.WriteString(fmt.Sprintf("%s\t%.5f\n", state_to_string(BoardState(pos)), game[pos]))
	}
}

func save_skips_table(skips []PrecalcSkip, game []float64, filename string) {
	f, err := os.Create(filename)
	if err != nil {
		panic("can't create output file")
	}

	buffer := bufio.NewWriter(f)
	defer func() {
		buffer.Flush()
		f.Close()
	}()

	for pos, skip_op := range skips {
		if skip_op.moves_to_skip != nil {

			moves_to_skip := make([]Move, 0)
			for _, m := range skip_op.moves_to_skip {
				if m[0] <= m[1] { // don't print move twice with it's mirror
					moves_to_skip = append(moves_to_skip, m)
				}
			}
			slices.Reverse(moves_to_skip)

			buffer.WriteString(fmt.Sprintf(
				"%s\tskip moves:%v; speedup:%.4g=>%.4g\n",
				state_to_string(BoardState(pos)), moves_to_skip, game[pos]+skip_op.speedup, game[pos],
			))
		}
	}
}

func main() {
	strategy := StrategyOptions{true, true}
	skips := make([]PrecalcSkip, 1<<12)
	game := calc_game_table(strategy, &skips)
	save_game_table(game, "game_optimal[do_skips].txt")
	save_skips_table(skips, game, "do_skips.txt")
}
