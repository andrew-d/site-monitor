var webpack = require('webpack');

var env = ["NODE_ENV", "NODE_DEBUG"].reduce(function(accum, k) {
    accum[k] = JSON.stringify(process.env[k] || "");
    return accum;
}, {});

module.exports = {
    cache: true,
    entry: "./static/js/App.jsx",
    output: {
        path: __dirname + "/build/js",
        filename: "bundle.js"
    },
    devtool: "source-map",
    module: {
        loaders: [
            //{ test: /\.less$/, loader: "style!css!less" },
            { test: /\.jsx$/, loader: "jsx-loader" },
            { test: /\.json$/, loader: "json" }
        ]
    },
    plugins: [
        new webpack.DefinePlugin({
            "process.env": env,
        })
    ],
};
