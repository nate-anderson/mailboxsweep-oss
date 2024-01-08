const withPWA = require('next-pwa')({
  dest: 'public',
  disable: !['staging', 'prod'].includes(process.env.ENV),
});
const { SubresourceIntegrityPlugin } = require('webpack-subresource-integrity');

/** @type {import('next').NextConfig} */
const nextConfig = /*withPWA*/({
  reactStrictMode: true,
  output: 'export',
  env: {
    MAILBOXSWEEP_API_HOST: process.env.MAILBOXSWEEP_API_HOST, 
    MAILBOXSWEEP_API_PORT: process.env.MAILBOXSWEEP_API_PORT,
    MAILBOXSWEEP_API_PROTO: process.env.MAILBOXSWEEP_API_PROTO,
    ENV: process.env.ENV
  },
  images:{
    unoptimized: true
  },
  trailingSlash: true,
  webpack: (config, { buildId, dev }) => {
    config.module.rules.push({
      test: /\.ya?ml$/,
      use: 'yaml-loader'
    });
    config.output.crossOriginLoading = "anonymous";
    config.plugins.push(
      new SubresourceIntegrityPlugin({
        hashFuncNames: ["sha256", "sha384"],
        enabled: true,
      })
    )
    return config;
  }
});

module.exports = nextConfig;
