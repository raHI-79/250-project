package store

import (
	"context"
	"fmt"
)

const PlatformAndroid = "android"

type FCMToken struct {
	ID       int64
	UserID   int64
	Token    string
	Platform string
	DeviceID string
}

const (
	saveFCMTokenSQL = `INSERT INTO fcm_tokens (user_id, token, platform, device_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (token) DO UPDATE SET user_id = EXCLUDED.user_id, platform = EXCLUDED.platform,
    device_id = EXCLUDED.device_id, last_used_at = now()
RETURNING id`
	deleteFCMTokenSQL = `DELETE FROM fcm_tokens WHERE user_id = $1 AND token = $2`
	dropFCMTokenSQL   = `DELETE FROM fcm_tokens WHERE token = $1`
	countFCMTokensSQL = `SELECT COUNT(*) FROM fcm_tokens WHERE user_id = $1`
	listFCMTokensSQL  = `SELECT id, user_id, token, platform, COALESCE(device_id, '')
FROM fcm_tokens WHERE user_id = ANY($1)`
	touchFCMTokensSQL = `UPDATE fcm_tokens SET last_used_at = now() WHERE token = ANY($1)`
)

func (s *Store) SaveFCMToken(ctx context.Context, token FCMToken) (int64, error) {
	platform := token.Platform
	if platform == "" {
		platform = PlatformAndroid
	}
	var id int64
	err := s.pool.QueryRow(ctx, saveFCMTokenSQL, token.UserID, token.Token, platform, token.DeviceID).
		Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("store: could not save the device token: %w", err)
	}
	return id, nil
}

func (s *Store) DeleteFCMToken(ctx context.Context, userID int64, token string) error {
	if _, err := s.pool.Exec(ctx, deleteFCMTokenSQL, userID, token); err != nil {
		return fmt.Errorf("store: could not delete the device token: %w", err)
	}
	return nil
}

func (s *Store) DropFCMToken(ctx context.Context, token string) error {
	if _, err := s.pool.Exec(ctx, dropFCMTokenSQL, token); err != nil {
		return fmt.Errorf("store: could not drop the device token: %w", err)
	}
	return nil
}

func (s *Store) CountFCMTokens(ctx context.Context, userID int64) (int, error) {
	var count int
	if err := s.pool.QueryRow(ctx, countFCMTokensSQL, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("store: could not count the device tokens: %w", err)
	}
	return count, nil
}

func (s *Store) ListFCMTokens(ctx context.Context, userIDs []int64) ([]FCMToken, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, listFCMTokensSQL, userIDs)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the device tokens: %w", err)
	}
	defer rows.Close()
	tokens := make([]FCMToken, 0)
	for rows.Next() {
		var token FCMToken
		err := rows.Scan(&token.ID, &token.UserID, &token.Token, &token.Platform, &token.DeviceID)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a device token: %w", err)
		}
		tokens = append(tokens, token)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the device tokens: %w", err)
	}
	return tokens, nil
}

func (s *Store) TouchFCMTokens(ctx context.Context, tokens []string) error {
	if len(tokens) == 0 {
		return nil
	}
	if _, err := s.pool.Exec(ctx, touchFCMTokensSQL, tokens); err != nil {
		return fmt.Errorf("store: could not touch the device tokens: %w", err)
	}
	return nil
}
