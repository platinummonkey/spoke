#!/bin/bash

# Install dependencies
npm install

# Create necessary directories if they don't exist
mkdir -p src/{components,pages,api,types}

# Start the development server
npm run dev 