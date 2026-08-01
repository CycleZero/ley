import i18n from "i18next";
import LanguageDetector from "i18next-browser-languagedetector";
import { initReactI18next } from "react-i18next";
import zhCN from "@/i18n/locales/zh-CN.json";
import enUS from "@/i18n/locales/en-US.json";

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      "zh-CN": { translation: zhCN },
      "en-US": { translation: enUS },
    },
    fallbackLng: "zh-CN",
    detection: {
      order: ["cookie", "localStorage", "navigator"],
      caches: ["cookie"],
      lookupCookie: "ley_lang",
    },
    interpolation: {
      escapeValue: false,
    },
  });

export default i18n;
