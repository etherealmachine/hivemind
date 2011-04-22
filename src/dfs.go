package main

func DFS(vertex int, board []byte, adjacent [][]int, out chan int) {
	visited := make([]bool, len(board))
	doDFS(vertex, board, adjacent, visited, out)
	out <- -1
}

func doDFS(vertex int, board[] byte, adjacent [][]int, visited []bool, out chan int) {
	out <- vertex
	visited[vertex] = true
	for _, n := range(adjacent[vertex]) {
		if n != -1 && board[n] == board[vertex] && !visited[n] {
			doDFS(n, board, adjacent, visited, out)
		} else if n != -1 && board[n] == EMPTY && !visited[n] {
			out <- n
		}
	}
}
