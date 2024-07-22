import { createRouter, createWebHashHistory } from 'vue-router';
import PageAbout from '~/components/PageAbout.vue';
import PageHome from '~/components/PageHome.vue';
import PageConnectInfoItems from '~/components/PageConnectInfoItems.vue';
import PageCustomText from '~/components/PageCustomText.vue';
import PageCustomReply from '~/components/mod/PageCustomReply.vue';
import PageJs from '~/components/mod/PageJs.vue';
import PageMiscDeck from '~/components/mod/PageMiscDeck.vue';
import PageHelpDoc from "~/components/mod/PageHelpDoc.vue";
import PageStory from '~/components/mod/PageStory.vue';
import PageCensor from '~/components/mod/PageCensor.vue';
import PageTest from '~/components/tool/PageTest.vue';
import PageResource from '~/components/tool/PageResource.vue';
import PageMiscSettings from '~/components/misc/PageMiscSettings.vue';
import PageMiscBackup from '~/components/misc/PageMiscBackup.vue';
import PageMiscGroup from '~/components/misc/PageMiscGroup.vue';
import PageMiscBan from '~/components/misc/PageMiscBan.vue';
import PageMiscAdvancedSettings from '~/components/misc/PageMiscAdvancedSettings.vue';

const router = createRouter({
  history: createWebHashHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/home', name: 'home', component: PageHome },
    { path: '/connect', component: PageConnectInfoItems },
    { path: '/custom-text/:category', component: PageCustomText, props: true },
    {
      path: '/mod',
      children: [
        { path: 'js', component: PageJs },
        { path: 'reply', component: PageCustomReply },
        { path: 'deck', component: PageMiscDeck },
        { path: 'helpdoc', component: PageHelpDoc },
        { path: 'story', component: PageStory },
        { path: 'censor', component: PageCensor },
      ]
    },
    {
      path: '/tool',
      children: [
        { path: 'test', component: PageTest },
        { path: 'resource', component: PageResource },
      ]
    },
    {
      path: '/misc',
      children: [
        { path: 'base-setting', component: PageMiscSettings },
        { path: 'backup', component: PageMiscBackup },
        { path: 'group', component: PageMiscGroup },
        { path: 'ban', component: PageMiscBan },
        { path: 'advanced-setting', component: PageMiscAdvancedSettings },
      ]
    },
    { path: '/about', component: PageAbout },
    { path: '/:catchAll(.*)', name: 'default', redirect: { name: 'home' } },
  ]
})

export default router