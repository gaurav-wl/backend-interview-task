SELECT 'CREATE DATABASE explore' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'explore');
