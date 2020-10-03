package domain

import (
	"context"
	"database/sql"

	pkgerr "github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/models"
	"github.com/Bnei-Baruch/gxydb-api/pkg/sqlutil"
)

type RoomStatisticsManager struct {
	db common.DBInterface
}

func NewRoomStatisticsManager(db common.DBInterface) *RoomStatisticsManager {
	return &RoomStatisticsManager{
		db: db,
	}
}

func (m *RoomStatisticsManager) OnAir(roomID int64) error {
	roomStats, err := m.getOrCreate(roomID)
	if err != nil {
		return pkgerr.Wrap(err, "getOrCreate")
	}

	roomStats.OnAir++

	err = m.update(roomStats)
	if err != nil {
		return pkgerr.Wrap(err, "update")
	}

	return nil
}

func (m *RoomStatisticsManager) GetAll() ([]*models.RoomStatistic, error) {
	return models.RoomStatistics(qm.Load(models.RoomStatisticRels.Room)).All(m.db)
}

func (m *RoomStatisticsManager) Reset(ctx context.Context) error {
	return sqlutil.InTx(ctx, m.db, func(tx *sql.Tx) error {
		rowsAff, err := models.RoomStatistics().DeleteAll(tx)
		if err != nil {
			return pkgerr.WithStack(err)
		}

		log.Ctx(ctx).Info().Int64("deleted", rowsAff).Msg("delete rooms statistics")

		return nil
	})
}

func (m *RoomStatisticsManager) getOrCreate(roomID int64) (*models.RoomStatistic, error) {
	var roomStats *models.RoomStatistic

	err := sqlutil.InTx(context.TODO(), m.db, func(tx *sql.Tx) error {
		var err error
		roomStats, err = models.FindRoomStatistic(tx, roomID)
		if err != nil && err != sql.ErrNoRows {
			return pkgerr.WithStack(err)
		}

		if roomStats != nil {
			return nil
		}

		roomStats = &models.RoomStatistic{
			RoomID: roomID,
		}
		err = roomStats.Insert(tx, boil.Infer())
		if err != nil {
			return pkgerr.WithStack(err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return roomStats, nil
}

func (m *RoomStatisticsManager) update(roomStats *models.RoomStatistic) error {
	return sqlutil.InTx(context.TODO(), m.db, func(tx *sql.Tx) error {
		_, err := roomStats.Update(tx, boil.Infer())
		if err != nil {
			return pkgerr.WithStack(err)
		}
		return nil
	})
}
