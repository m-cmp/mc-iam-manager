const path = require('path');
const fs = require('fs');
const { CleanWebpackPlugin } = require('clean-webpack-plugin');

function getAllFiles(dirPath, arrayOfFiles) {
    const files = fs.readdirSync(dirPath);

    arrayOfFiles = arrayOfFiles || [];

    files.forEach(function(file) {
        if (fs.statSync(path.join(dirPath, file)).isDirectory()) {
            arrayOfFiles = getAllFiles(path.join(dirPath, file), arrayOfFiles);
        } else if (file.endsWith('.js')) {
            arrayOfFiles.push(path.join(dirPath, file));
        }
    });

    return arrayOfFiles;
}

const entryDir = path.resolve(__dirname, 'js');
const entries = getAllFiles(entryDir)
    .reduce((entries, file) => {
        const relativePath = path.relative(entryDir, file); 
        const name = relativePath.replace(/\.js$/, ''); 
        entries[name] = file;
        return entries;
    }, {});

module.exports = {
    entry: entries,
    output: {
        filename: '[name].bundle.js', 
        path: path.resolve(__dirname, 'assets'),
    },
    mode: 'production',
    plugins: [
        new CleanWebpackPlugin(), 
    ],
};
