const path = require('path');
const CopyPlugin = require('copy-webpack-plugin');
const HtmlWebpackPlugin = require('html-webpack-plugin');

module.exports = {
  entry: {
    background: './background.js',
    content: './content.js',
    popup: './popup.js',
    planSelector: './planSelector.js',
    api: './api.js',
    utils: './utils.js'
  },
  output: {
    path: path.resolve(__dirname, 'dist'),
    filename: '[name].js',
    clean: true
  },
  module: {
    rules: [
      {
        test: /\\.js$/,
        exclude: /node_modules/,
        use: {
          loader: 'babel-loader',
          options: {
            presets: ['@babel/preset-env']
          }
        }
      },
      {
        test: /\\.css$/,
        use: ['style-loader', 'css-loader']
      }
    ]
  },
  plugins: [
    new CopyPlugin({
      patterns: [
        { from: 'manifest.json', to: 'manifest.json' },
        { from: 'popup.css', to: 'popup.css' },
        { from: 'planSelector.css', to: 'planSelector.css' },
        { 
          from: 'icons', 
          to: 'icons',
          noErrorOnMissing: true
        }
      ]
    }),
    new HtmlWebpackPlugin({
      template: './popup.html',
      filename: 'popup.html',
      chunks: ['popup']
    }),
    new HtmlWebpackPlugin({
      template: './planSelector.html',
      filename: 'planSelector.html',
      chunks: ['planSelector']
    })
  ],
  optimization: {
    splitChunks: {
      chunks: 'all'
    }
  }
};
