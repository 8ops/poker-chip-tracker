export interface Room {
  id: string
  roomCode: string
  name: string
  status: 'OPEN' | 'CLOSED'
  createdAt: string
}

export interface Player {
  id: string
  roomId: string
  name: string
  initialChip: number
  currentChip: number
  totalChange?: number
}

export interface GameRound {
  id: string
  roomId: string
  roundNo: number
}

export interface ChipRecord {
  id: string
  roundId: string
  playerId: string
  playerName?: string
  changeAmount: number
}

export interface RoundView {
  id: string
  roundNo: number
  records: ChipRecord[]
}

export interface RoomDetail {
  room: Room
  players: Player[]
  rounds: RoundView[]
}

export interface PlayerStat {
  id: string
  name: string
  initialChip: number
  currentChip: number
  totalChange: number
  rank: number
}

export interface History {
  room: Room
  totalRounds: number
  closedAt?: string
  stats: PlayerStat[]
  rounds: RoundView[]
}

export interface ApiError {
  code: number
  message: string
}
