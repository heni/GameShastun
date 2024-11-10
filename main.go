package main

import (
	"bufio"
	"fmt"
	"os"
	"slices"
)

type BoardState int
type Move [2]int

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

func calc_game_table() []float64 {
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
		d, t := 1.0, 1.0
		for a := 1; a <= 6; a++ {
			for b := 1; b <= 6; b++ {
				s1, ok1 := update_state_1(BoardState(pos), Move{a, b})
				s2, ok2 := update_state_2(BoardState(pos), Move{a, b})
				if ok1 && ok2 {
					t += min(time_to_win[int(s1)], time_to_win[int(s2)]) / 36.0
					// t += (.5*time_to_win[int(s1)] + .5*time_to_win[int(s2)]) / 36.0
				} else if ok1 {
					t += time_to_win[int(s1)] / 36.0
				} else if ok2 {
					t += time_to_win[int(s2)] / 36.0
				} else {
					d -= 1 / 36.0
				}
			}
		}
		time_to_win[pos] = t / d
	}

	return time_to_win
}

func save_game_table(game []float64, filename string) {
	f, err := os.Create(filename)
	if err != nil {
		panic("can't create output file")
	}

	defer f.Close()
	buffer := bufio.NewWriter(f)
	for pos := range game {
		buffer.WriteString("{")
		for p := 1; p <= 12; p++ {
			if pos&(1<<(p-1)) != 0 {
				if pos >= 1<<p {
					buffer.WriteString(fmt.Sprintf("%d,", p))
				} else {
					buffer.WriteString(fmt.Sprintf("%d", p))
				}
			}
		}
		buffer.WriteString(fmt.Sprintf("}\t%.5f\n", game[pos]))
	}
	buffer.Flush()
}

func main() {
	game := calc_game_table()
	save_game_table(game, "game_optimal.txt")
}
