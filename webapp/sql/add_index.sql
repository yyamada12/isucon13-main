CREATE INDEX idx_reactions_livestream_id ON reactions (livestream_id);
CREATE INDEX idx_ng_words_user_id_livestream_id ON ng_words (user_id, livestream_id);