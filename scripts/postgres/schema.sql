-- Enable UUID extension for generating user IDs
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create user role enum type
CREATE TYPE user_role AS ENUM (
  'USER',
  'ADMIN'
);

-- Create users table
CREATE TABLE IF NOT EXISTS users(
	id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
	username TEXT UNIQUE NOT NULL,
	email TEXT UNIQUE NOT NULL,
	password_hash TEXT NOT NULL,
	role user_role NOT NULL DEFAULT 'USER',
	created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now()
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Add comments for documentation
COMMENT ON TABLE users IS 'User accounts for the Pomodoro service';
COMMENT ON COLUMN users.role IS 'User role for access control: USER or ADMIN';


