package health

import "github.com/jackc/pgx/v5/pgxpool"

// PoolStats holds a snapshot of connection pool metrics for health reporting.
type PoolStats struct {
	AcquiredConns int32 `json:"acquired_conns"`
	IdleConns     int32 `json:"idle_conns"`
	MaxConns      int32 `json:"max_conns"`
}

func poolStatsFromPgx(stat *pgxpool.Stat) PoolStats {
	if stat == nil {
		return PoolStats{}
	}
	return PoolStats{
		AcquiredConns: stat.AcquiredConns(),
		IdleConns:     stat.IdleConns(),
		MaxConns:      stat.MaxConns(),
	}
}
