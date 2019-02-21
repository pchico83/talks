INSERT INTO users(id, created_at, updated_at, email, token, verified) VALUES('799273b1-b067-4f84-b632-864c543c4dc1', now(), now(), 'bot@okteto.com', 'fake-token-2', 't') ON CONFLICT DO NOTHING;
