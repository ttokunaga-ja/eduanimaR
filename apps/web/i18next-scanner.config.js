module.exports = {
  input: [
    'src/**/*.{js,jsx,ts,tsx,md}',
  ],
  output: './public/locales/$LOCALE/$NAMESPACE.json',
  options: {
    debug: false,
    func: {
      list: ['t', 'i18n.t'],
      extensions: ['.js', '.jsx', '.ts', '.tsx'],
    },
    lngs: ['en', 'ja'],
    ns: ['common'],
    defaultLng: 'en',
    defaultNs: 'common',
    resource: {
      loadPath: 'public/locales/{{lng}}/{{ns}}.json',
      savePath: 'public/locales/{{lng}}/{{ns}}.json',
    },
    interpolation: {
      prefix: '{{',
      suffix: '}}',
    },
  },
};
