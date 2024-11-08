const path = require('path');

module.exports = {
  target: 'node',
  mode: 'none',
  entry: './src/extension.ts',
  output: {
    path: path.resolve(__dirname, 'dist'),
    filename: 'extension.js',
    libraryTarget: 'commonjs2'
  },
  externals: {
    vscode: 'commonjs vscode'
  },
  resolve: {
    extensions: ['.ts', '.js']
  },
  module: {
    rules: [
      {
        test: /\.ts$/,
        exclude: /node_modules/,
        use: [
          {
            loader: 'ts-loader',
            options: {
              compilerOptions: {
                "module": "es6",
                "moduleResolution": "node"
              }
            }
          }
        ]
      }
    ]
  },
  devtool: 'nosources-source-map',
  stats: {
    assets: true,
    modules: true,
    errors: true,
    errorDetails: true,
    logging: 'verbose'
  },
  infrastructureLogging: {
    level: 'verbose',
    debug: true
  }
};
