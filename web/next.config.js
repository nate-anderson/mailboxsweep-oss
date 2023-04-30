/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true, // set this to true for production,
  output: 'standalone',
  env: {
    MAILBOXSWEEP_API_HOST: process.env.MAILBOXSWEEP_API_HOST, 
    MAILBOXSWEEP_API_PORT: process.env.MAILBOXSWEEP_API_PORT
  }
};

module.exports = nextConfig;
