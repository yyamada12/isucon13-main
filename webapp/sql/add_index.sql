CREATE INDEX idx_reactions_livestream_id_created_at ON reactions (livestream_id, created_at DESC);
CREATE INDEX idx_ng_words_user_id_livestream_id ON ng_words (user_id, livestream_id);
CREATE INDEX idx_icons_user_id ON icons (user_id);
CREATE INDEX idx_livestreams_user_id ON livestreams (user_id);
CREATE INDEX idx_slots_start_at_end_at ON reservation_slots (start_at, end_at);
CREATE INDEX idx_livecomments_livestream_id_created_at ON livecomments (livestream_id, created_at DESC);