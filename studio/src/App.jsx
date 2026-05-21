import { useEffect, useMemo, useRef, useState } from 'react';
import {
  BarChart3,
  BookImage,
  Brush,
  ChevronDown,
  Compass,
  Copy,
  Download,
  Flame,
  Home,
  Hash,
  Image as ImageIcon,
  LayoutTemplate,
  LoaderCircle,
  LogOut,
  Mail,
  Menu,
  Moon,
  Palette,
  Plus,
  RefreshCw,
  Search,
  Settings,
  Sparkles,
  Star,
  Sun,
  Languages,
  UserRound,
  Video,
  WandSparkles,
  X,
  Zap,
} from 'lucide-react';

const API_BASE_URL = 'http://127.0.0.1:8080/pk';
const AUTH_APP_ID = 'berserk.web';
const AUTH_STORAGE_KEY = 'berserk-ai-auth-session';
const STYLE_FAVORITES_KEY = 'berserk-ai-style-favorites';

const navItems = [
  { label: '主页', icon: Home, view: 'home' },
  { label: '收藏', icon: Star, view: 'favorites' },
  { label: '生成记录', icon: RefreshCw, view: 'history' },
  { label: '个人中心', icon: Settings, view: 'profile' },
];

const aiAppItems = [
  { label: '角色创建器', icon: UserRound },
  { label: 'AI 动漫生成器', icon: ImageIcon },
  { label: '线稿上色', icon: Palette },
  { label: 'AI 动画制作工具', icon: Video },
];
const filterChips = [
  { label: '所有帖子' },
  { label: '精选', active: true, icon: Star },
  { label: '效果', icon: LayoutTemplate },
  { label: '动画片', icon: Video },
  { label: '音乐', icon: Sparkles },
  { label: '幻灯片', icon: BookImage },
  { label: 'BerserkAIConfession', featured: true },
  { label: '场景', hash: true },
  { label: 'OC', hash: true },
  { label: '可爱' },
  { label: '毛茸茸的' },
  { label: '船' },
  { label: '老婆' },
  { label: '丈夫' },
  { label: '同性爱' },
  { label: '女同性爱' },
  { label: 'NSFW' },
  { label: '原神' },
];

const generationSizes = ['自动', '1:1', '3:4', '4:5', '2:3', '9:16', '4:3', '5:4', '3:2', '16:9', '21:9'];

const styleCategories = [
  { id: 'favorites', label: '收藏' },
  { id: 'art', label: '艺术' },
  { id: 'meme', label: '梗图' },
  { id: 'painterly', label: '绘画感' },
  { id: 'chibi', label: 'Q版' },
  { id: 'male', label: '男性向' },
  { id: 'anime', label: '动漫' },
  { id: 'manga', label: '漫画' },
  { id: 'sketch', label: '素描' },
  { id: 'furry', label: '毛茸茸' },
  { id: '3d', label: '3D' },
  { id: 'flat', label: '扁平' },
  { id: 'general', label: '通用' },
  { id: 'custom', label: '自定义' },
];

const stylePresets = [
  { name: '鲜艳动漫风', category: 'art', image: 'https://komiko.app/images/kusa_styles/vibrant_anime.webp' },
  { name: '高反差亮面风', category: 'art', image: 'https://komiko.app/images/kusa_styles/high_contrast_glossy.webp' },
  { name: '漆面插画', category: 'art', image: 'https://komiko.app/images/kusa_styles/lacquered_illustration.webp' },
  { name: '半写实肖像', category: 'art', image: 'https://komiko.app/images/kusa_styles/semi_realistic_portrait.webp' },
  { name: '柔和粉彩', category: 'painterly', image: 'https://komiko.app/images/kusa_styles/soft_pastel.webp' },
  { name: '褪色画布', category: 'painterly', image: 'https://dihulvhqvmoxyhkxovko.supabase.co/storage/v1/object/public/husbando-land/assets/kusa_styles/faded-canvas-style.png' },
  { name: '柔光插画', category: 'painterly', image: 'https://komiko.app/images/kusa_styles/soft_light_illustration.webp' },
  { name: '晶体锐边', category: 'anime', image: 'https://dihulvhqvmoxyhkxovko.supabase.co/storage/v1/object/public/husbando-land/assets/kusa_styles/crystal-edge-style.png' },
  { name: '虹彩质感', category: 'anime', image: 'https://komiko.app/images/kusa_styles/iridescent_new.webp' },
  { name: '水彩插画', category: 'painterly', image: 'https://dihulvhqvmoxyhkxovko.supabase.co/storage/v1/object/public/husbando-land/assets/watercolor_Illustration.webp' },
  { name: '高光插画', category: 'anime', image: 'https://komiko.app/images/kusa_styles/high_gloss_illustration.webp' },
  { name: '甜系粉彩', category: 'chibi', image: 'https://komiko.app/images/kusa_styles/sweet_pastel.webp' },
  { name: '闪耀插画', category: 'anime', image: 'https://komiko.app/images/kusa_styles/dazzling_illustration.webp' },
  { name: '柔和阴影', category: 'manga', image: 'https://komiko.app/images/kusa_styles/soft_shading_new.webp' },
  { name: '低饱和插画', category: 'manga', image: 'https://komiko.app/images/kusa_styles/desaturated_illustration_new.webp' },
  { name: '亮面动漫', category: 'anime', image: 'https://komiko.app/images/kusa_styles/glossy_anime_new.webp' },
  { name: '干净线稿', category: 'sketch', image: 'https://komiko.app/images/kusa_styles/clean_lines.webp' },
  { name: '流行卡通', category: 'flat', image: 'https://komiko.app/images/kusa_styles/pop_toon_style.webp' },
  { name: '柔和像素', category: '3d', image: 'https://komiko.app/images/kusa_styles/soft_pixel_art.webp' },
  { name: '氛围发光', category: 'general', image: 'https://komiko.app/images/kusa_styles/moody_glow_style.webp' },
  { name: '柔萌阴影', category: 'chibi', image: 'https://komiko.app/images/kusa_styles/soft_shaded_moe_style.webp' },
  { name: '赛博糖果', category: 'meme', image: 'https://komiko.app/images/kusa_styles/cyber_candy.webp' },
  { name: '幻想龙族', category: 'furry', image: 'https://dihulvhqvmoxyhkxovko.supabase.co/storage/v1/object/public/husbando-land/assets/kusa_styles/fantasy-dragon-style.webp' },
  { name: '男性奇幻', category: 'male', image: 'https://komiko.app/images/kusa_styles/katsuya_terada_inspired_fantasy_art.webp' },
  { name: '自定义示例', category: 'custom', image: 'https://komiko.app/images/kusa_styles/mischiefstyle.webp' },
];

const creditPackages = [
  {
    id: 'credits_trial',
    name: '限时体验包',
    price: '¥1',
    credits: '10 积分',
    icon: '/pricing-icons/package-trial.png',
    tone: 'blue',
    features: ['限时体验专享', '可生成约 2 次基础模型图片', '适合测试出图流程', '购买后立即到账'],
  },
  {
    id: 'credits_100',
    name: '灵感入门包',
    price: '¥10',
    credits: '100 积分',
    icon: '/pricing-icons/package-100.png',
    tone: 'blue',
    features: ['适合轻量试用', '可生成约 20 次图片', '购买后立即到账', '积分长期保留'],
  },
  {
    id: 'credits_500',
    name: '创作加速包',
    price: '¥49',
    credits: '500 积分',
    icon: '/pricing-icons/package-500.png',
    popular: true,
    tone: 'purple',
    features: ['适合日常创作', '可生成约 100 次图片', '比入门包更划算', '购买后立即到账'],
  },
  {
    id: 'credits_1000',
    name: '高频创作包',
    price: '¥95',
    credits: '1,000 积分',
    icon: '/pricing-icons/package-1000.png',
    tone: 'gold',
    features: ['适合高频出图', '可生成约 200 次图片', '批量探索不同风格', '购买后立即到账'],
  },
];

const pricingFaqs = [
  { question: '积分如何消耗？', answer: '不同模型会按配置消耗不同积分，基础模型通常每次生成消耗 5 积分。实际消耗会在生成按钮和模型选择处展示。' },
  { question: '购买后多久到账？', answer: '如果是站内直接购买会立即到账；通过卡密平台购买时，拿到卡号和密码后在订阅页兑换，兑换成功后积分立即进入当前账号。' },
  { question: '积分会过期吗？', answer: '当前积分长期保留，不会按月清零。后续如果调整有效期，会提前在页面和公告里说明。' },
  { question: '支持哪些支付方式？', answer: '支付方式由配置的卡密平台决定。平台支付完成后会提供卡号和密码，你回到这里兑换即可。' },
  { question: '生成失败会退还积分吗？', answer: '会。后端任务失败时会按本次任务消耗自动退回积分，避免因为接口或模型异常造成损失。' },
  { question: '可以多次购买积分包吗？', answer: '可以。积分包可以重复购买和兑换，多次兑换的积分会累加到同一个账号余额中。' },
];

const defaultImageModels = [
  { id: 'gpt-image', name: 'GPT Image', provider: 'OpenAI', creditCost: 5 },
  { id: 'seedream', name: 'Seedream', provider: 'ByteDance', creditCost: 6 },
  { id: 'qwen-image', name: 'Qwen Image', provider: 'Alibaba', creditCost: 5 },
];

function tagsFromImages(items) {
  const source = items
    .slice(0, 24)
    .flatMap((item) => [item.title, item.promptZh, item.style, item.model])
    .join(' ');
  const dictionary = [
    ['海报', /海报|广告|品牌|商业/],
    ['产品图', /产品|手袋|饮料|香水|包装/],
    ['动漫', /动漫|角色|插画|二次元/],
    ['人物', /人物|女性|男性|模特|肖像/],
    ['机甲', /机甲|机械|未来/],
    ['风景', /风景|山|城市|自然|街头/],
    ['建筑', /建筑|空间|室内/],
    ['可爱', /可爱|猫|萌|Q版/],
    ['幻想', /奇幻|魔法|梦幻|冒险/],
    ['赛博朋克', /赛博|霓虹|未来感/],
  ];
  const tags = dictionary.filter(([, pattern]) => pattern.test(source)).map(([label]) => label);
  return Array.from(new Set(['小说封面', '所有帖子', '精选', ...tags, 'BerserkAIConfession', 'OC', 'NSFW'])).slice(0, 18);
}

function normalizeGalleryItem(item) {
  const prompt = item.prompt || '';
  const style = item.style || item.tag || '作品';
  return {
    id: item.id,
    title: style,
    promptZh: prompt,
    src: item.thumbnailURL || item.image,
    fullSrc: item.image,
    width: 1024,
    height: item.ratio === 'landscape' ? 768 : item.ratio === 'square' ? 1024 : 1365,
    author: item.author || 'Berserk AI',
    authorAvatarURL: item.authorAvatarURL || '/assets/berserk-ai-icon.png',
    likes: item.likeCount || 0,
    likeCount: item.likeCount || 0,
    likedByMe: Boolean(item.likedByMe),
    favoritedByMe: Boolean(item.favoritedByMe),
    favoriteCount: item.favoriteCount || 0,
    modelID: item.modelID || '',
    model: item.modelName || item.model || 'GPT Image',
    style,
    isFeatured: Boolean(item.isFeatured),
    isPromptFeatured: Boolean(item.isPromptFeatured),
    createdAt: item.createdAt,
  };
}

function modelIconFor(model) {
  const id = String(model?.id || model?.modelID || '').toLowerCase();
  const provider = String(model?.provider || model?.model || model?.name || '').toLowerCase();
  if (id.includes('gemini') || provider.includes('google') || provider.includes('gemini')) return 'https://cdn.simpleicons.org/google/4285F4';
  if (id.includes('qwen') || provider.includes('alibaba') || provider.includes('aliyun')) return 'https://cdn.simpleicons.org/alibabacloud/FF6A00';
  if (id.includes('seed') || provider.includes('byte') || provider.includes('doubao')) return 'https://cdn.simpleicons.org/bytedance/111111';
  return '/assets/openai-logo.svg';
}

function App() {
  const [authOpen, setAuthOpen] = useState(false);
  const [selectedImage, setSelectedImage] = useState(null);
  const [authSession, setAuthSession] = useState(() => readStoredAuthSession());
  const [theme, setTheme] = useState('light');
  const [view, setView] = useState('home');
  const [profileOpen, setProfileOpen] = useState(false);
  const [feedItems, setFeedItems] = useState([]);
  const [feedLoading, setFeedLoading] = useState(true);
  const [feedError, setFeedError] = useState('');
  const [galleryQuery, setGalleryQuery] = useState('');
  const [imageModels, setImageModels] = useState(defaultImageModels);
  const [packageItems, setPackageItems] = useState(creditPackages);
  const [comingSoonOpen, setComingSoonOpen] = useState(false);
  const [generationTasks, setGenerationTasks] = useState([]);
  const [taskNotice, setTaskNotice] = useState('');
  const [appModal, setAppModal] = useState(null);
  const [galleryRefreshKey, setGalleryRefreshKey] = useState(0);
  const previousActiveTaskCountRef = useRef(0);
  const pendingTaskCount = generationTasks.filter((task) => ['queued', 'running'].includes(task.status)).length;
  const galleryAuthKey = view === 'favorites' ? authSession?.token || '' : '';

  useEffect(() => {
    getJSON('/api/v1/images/models')
      .then((payload) => {
        if (Array.isArray(payload?.items) && payload.items.length > 0) setImageModels(payload.items);
      })
      .catch(() => {});
    getJSON('/api/v1/credits/packages')
      .then((payload) => {
        if (Array.isArray(payload?.items) && payload.items.length > 0) {
          setPackageItems(payload.items.map(normalizeCreditPackage));
        }
      })
      .catch(() => {});
  }, []);

  useEffect(() => {
    if (view === 'favorites' && !authSession?.token) {
      setFeedItems([]);
      setFeedLoading(false);
      setFeedError('请先登录后查看收藏。');
      setAuthOpen(true);
      return;
    }
    let cancelled = false;
    setFeedLoading(true);
    const favoriteQuery = view === 'favorites' ? '&favorite=true' : '';
    const searchQuery = galleryQuery ? `&q=${encodeURIComponent(galleryQuery)}` : '';
    getJSON(`/api/v1/images/gallery?limit=100${favoriteQuery}${searchQuery}`, authSession?.token)
      .then(async (payload) => {
        const nextItems = (payload?.items || []).map(normalizeGalleryItem);
        await preloadGalleryImages(nextItems.slice(0, 16).map((item) => item.src));
        if (cancelled) return;
        setFeedItems(nextItems);
        setFeedError('');
      })
      .catch((error) => {
        if (cancelled) return;
        setFeedError(getErrorMessage(error, '图库加载失败'));
        setFeedItems([]);
      })
      .finally(() => {
        if (!cancelled) setFeedLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [galleryAuthKey, view, galleryQuery, galleryRefreshKey]);

  const loadGenerationTasks = async () => {
    if (!authSession?.token) {
      setGenerationTasks([]);
      previousActiveTaskCountRef.current = 0;
      return [];
    }
    const payload = await getJSON('/api/v1/images/tasks?limit=30', authSession.token);
    const items = Array.isArray(payload?.items) ? payload.items : [];
    const activeCount = items.filter((task) => ['queued', 'running'].includes(task.status)).length;
    if (previousActiveTaskCountRef.current > 0 && activeCount === 0) {
      setGalleryRefreshKey((value) => value + 1);
    }
    previousActiveTaskCountRef.current = activeCount;
    setGenerationTasks(items);
    return items;
  };

  useEffect(() => {
    if (!authSession?.token) {
      setGenerationTasks([]);
      previousActiveTaskCountRef.current = 0;
      return undefined;
    }
    let cancelled = false;
    const refresh = () => {
      loadGenerationTasks().catch(() => {
        if (!cancelled) setGenerationTasks([]);
      });
    };
    refresh();
    const timer = window.setInterval(refresh, pendingTaskCount > 0 ? 4500 : 12000);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [authSession?.token, pendingTaskCount]);

  useEffect(() => {
    if (!authSession?.token) return undefined;
    let cancelled = false;
    getJSON('/api/v1/me', authSession.token)
      .then((user) => {
        if (cancelled || !user?.id) return;
        setAuthSession((session) => {
          if (!session?.token) return session;
          const nextSession = { ...session, user };
          window.localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(nextSession));
          return nextSession;
        });
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [authSession?.token]);

  const handleAuthSuccess = (session) => {
    setAuthSession(session);
    window.localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(session));
    setAuthOpen(false);
  };

  const handleLogout = () => {
    setAuthSession(null);
    window.localStorage.removeItem(AUTH_STORAGE_KEY);
  };

  const handleSessionUser = (user) => {
    if (!authSession || !user) return;
    const nextSession = { ...authSession, user };
    setAuthSession(nextSession);
    window.localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(nextSession));
  };

  const handleGenerateImage = async ({
    prompt,
    style,
    size = '3:4',
    modelID,
    images = [],
    quality = 'medium',
    n = 1,
    negativePrompt = '',
    resolution = 'auto',
    lockedSeed = false,
  }) => {
    if (!authSession?.token) {
      setAuthOpen(true);
      return;
    }
    const payload = await authPostJSON('/api/v1/images/tasks', authSession.token, {
      prompt,
      style,
      size: sizeToBackendSize(size),
      quality,
      n,
      modelID,
      images,
      negativePrompt,
      resolution,
      lockedSeed,
    }, '生成失败');
    if (payload?.user) handleSessionUser(payload.user);
    if (payload?.task) {
      setTaskNotice('图片已进入生成队列，你可以在生成记录里查看进度。');
      setGenerationTasks((items) => [payload.task, ...items.filter((task) => task.id !== payload.task.id)]);
      previousActiveTaskCountRef.current = Math.max(1, previousActiveTaskCountRef.current);
      return;
    }
    const created = (payload?.images || []).map((image, index) => ({
      id: `generated-${Date.now()}-${index}`,
      title: prompt.slice(0, 24) || '新生成图片',
      promptZh: prompt,
      src: image.thumbnailURL || image.url,
      fullSrc: image.url,
      width: 1024,
      height: size.includes('16:9') || size.includes('4:3') || size.includes('21:9') ? 768 : 1365,
      author: authSession.user?.displayName || authSession.user?.email || 'Berserk AI',
      likes: 0,
      likeCount: 0,
      likedByMe: false,
      model: payload?.modelName || imageModels.find((model) => model.id === modelID)?.name || 'GPT Image',
      authorAvatarURL: authSession.user?.avatarURL || '/assets/berserk-ai-icon.png',
      isFeatured: false,
      isPromptFeatured: false,
    }));
    if (created.length > 0) setFeedItems((items) => [...created, ...items]);
  };

  const handleLikeImage = async (item, liked) => {
    setFeedItems((items) => items.map((candidate) => (candidate.id === item.id ? { ...candidate, likedByMe: liked, likeCount: Math.max(0, (candidate.likeCount || candidate.likes || 0) + (liked ? 1 : -1)) } : candidate)));
    if (!authSession?.token) {
      setAuthOpen(true);
      return;
    }
    if (!String(item.id).startsWith('generated-')) {
      authPostJSON(`/api/v1/images/gallery/${item.id}/like`, authSession.token, { liked }, '点赞失败')
        .then((payload) => {
          if (payload?.item) {
            const nextItem = normalizeGalleryItem(payload.item);
            setFeedItems((items) => items.map((candidate) => (candidate.id === nextItem.id ? nextItem : candidate)));
            setSelectedImage((current) => (current?.id === nextItem.id ? nextItem : current));
          }
        })
        .catch((error) => setAppModal({ tone: 'error', title: '点赞失败', message: getErrorMessage(error, '点赞失败') }));
    }
  };

  const handleFeatureImage = async (item, next) => {
    setFeedItems((items) => items.map((candidate) => (candidate.id === item.id ? { ...candidate, ...next } : candidate)));
    setSelectedImage((current) => (current?.id === item.id ? { ...current, ...next } : current));
    if (authSession?.token && !String(item.id).startsWith('generated-')) {
      authPatchJSON(`/api/v1/images/gallery/${item.id}/featured`, authSession.token, next, '精选失败').catch(() => {});
    }
  };

  const handleFavoriteImage = async (item, favorited) => {
    setFeedItems((items) => items.map((candidate) => (candidate.id === item.id ? { ...candidate, favoritedByMe: favorited, favoriteCount: Math.max(0, (candidate.favoriteCount || 0) + (favorited ? 1 : -1)) } : candidate)));
    setSelectedImage((current) => (current?.id === item.id ? { ...current, favoritedByMe: favorited, favoriteCount: Math.max(0, (current.favoriteCount || 0) + (favorited ? 1 : -1)) } : current));
    if (!authSession?.token) {
      setAuthOpen(true);
      return;
    }
    if (!String(item.id).startsWith('generated-')) {
      authPostJSON(`/api/v1/images/gallery/${item.id}/favorite`, authSession.token, { favorited }, '收藏失败')
        .then((payload) => {
          if (payload?.item) {
            const nextItem = normalizeGalleryItem(payload.item);
            setFeedItems((items) => items.map((candidate) => (candidate.id === nextItem.id ? nextItem : candidate)));
            setSelectedImage((current) => (current?.id === nextItem.id ? nextItem : current));
          }
        })
        .catch((error) => setAppModal({ tone: 'error', title: '收藏失败', message: getErrorMessage(error, '收藏失败') }));
    }
  };

  return (
    <div className={`berserk-app${view === 'pricing' ? ' pricing-mode' : ''}`} data-theme={theme}>
      {view === 'pricing' ? (
        <PricingPage packages={packageItems} authSession={authSession} onAuthOpen={() => setAuthOpen(true)} onUserChange={handleSessionUser} onBack={() => setView('home')} />
      ) : (
        <>
          <TopBar currentUser={authSession?.user} theme={theme} onThemeChange={setTheme} onAuthOpen={() => setAuthOpen(true)} />
          <Sidebar currentUser={authSession?.user} currentView={view} pendingTaskCount={pendingTaskCount} onNavigate={setView} onProfileOpen={() => setProfileOpen(true)} onAuthOpen={() => setAuthOpen(true)} onLogout={handleLogout} onComingSoon={() => setComingSoonOpen(true)} />
          <main className="workspace">
            <MobileTopbar onAuthOpen={() => setAuthOpen(true)} />
            {view === 'history' ? (
              <GenerationHistory tasks={generationTasks} onRefresh={loadGenerationTasks} />
            ) : (
              <>
                <KomikoComposer models={imageModels} feedItems={feedItems} activeQuery={galleryQuery} onQueryChange={setGalleryQuery} onGenerate={handleGenerateImage} />
                {feedError ? <p className="feed-error">{feedError}</p> : null}
                <MasonryFeed items={feedItems} loading={feedLoading} onOpen={setSelectedImage} onLike={handleLikeImage} onFeature={handleFeatureImage} onFavorite={handleFavoriteImage} />
              </>
            )}
          </main>
        </>
      )}
      {selectedImage ? <ImagePreview item={selectedImage} models={imageModels} currentUser={authSession?.user} onClose={() => setSelectedImage(null)} onLike={handleLikeImage} onFavorite={handleFavoriteImage} onGenerate={handleGenerateImage} onMessage={setAppModal} /> : null}
      {profileOpen ? <ProfileModal session={authSession} onClose={() => setProfileOpen(false)} onAuthOpen={() => setAuthOpen(true)} onUserChange={handleSessionUser} /> : null}
      {authOpen ? <AuthModal onClose={() => setAuthOpen(false)} onSuccess={handleAuthSuccess} /> : null}
      {comingSoonOpen ? <AppModal title="即将上线" message="AI 应用模块正在打磨中，后续会接入更多创作工具。" onClose={() => setComingSoonOpen(false)} /> : null}
      {taskNotice ? <AppModal title="正在生成" message={taskNotice} onClose={() => setTaskNotice('')} /> : null}
      {appModal ? <AppModal {...appModal} onClose={() => setAppModal(null)} /> : null}
    </div>
  );
}

function TopBar({ currentUser, theme, onThemeChange, onAuthOpen }) {
  const [themeMenuOpen, setThemeMenuOpen] = useState(false);

  return (
    <header className="topbar">
      <div className="topbar-left" />
      <div className="topbar-actions">
        <div className="theme-menu-wrap">
          <button type="button" aria-label="切换明暗模式" onClick={() => setThemeMenuOpen((open) => !open)}>
            {theme === 'dark' ? <Moon size={18} /> : <Sun size={18} />}
          </button>
          {themeMenuOpen ? (
            <div className="theme-popover" role="menu">
              <button
                className={theme === 'light' ? 'active' : ''}
                type="button"
                onClick={() => {
                  onThemeChange('light');
                  setThemeMenuOpen(false);
                }}
              >
                <Sun size={15} /> 日间
              </button>
              <button
                className={theme === 'dark' ? 'active' : ''}
                type="button"
                onClick={() => {
                  onThemeChange('dark');
                  setThemeMenuOpen(false);
                }}
              >
                <Moon size={15} /> 夜间
              </button>
            </div>
          ) : null}
        </div>
        <button className="top-login" type="button" onClick={onAuthOpen}>
          {currentUser ? '已登录' : '登录'}
        </button>
      </div>
    </header>
  );
}

function Sidebar({ currentUser, currentView, pendingTaskCount, onNavigate, onProfileOpen, onAuthOpen, onLogout, onComingSoon }) {
  return (
    <aside className="sidebar">
      <a className="brand" href="#" aria-label="Berserk AI" onClick={() => onNavigate('home')}>
        <img src="/assets/berserk-ai-icon.png" alt="" />
        <span>
          <strong>BERSERK AI</strong>
          <small>www.berserk-ai.com</small>
        </span>
      </a>
      <nav className="side-nav" aria-label="主导航">
        {navItems.map(({ label, icon: Icon, view: itemView, badge }) => (
          <a
            className={currentView === itemView ? 'active' : ''}
            href={itemView === 'home' ? '#' : '#inspiration-feed'}
            key={label}
            onClick={(event) => {
              if (itemView === 'profile') {
                event.preventDefault();
                onProfileOpen();
                return;
              }
              event.preventDefault();
              onNavigate(itemView);
            }}
          >
            <Icon size={18} />
            <span>{label}</span>
            {itemView === 'history' && pendingTaskCount > 0 ? <em>{pendingTaskCount}</em> : badge ? <em>{badge}</em> : null}
          </a>
        ))}
      </nav>
      <div className="side-section">
        <button type="button">
          <span>AI 应用</span>
          <i />
          <em>正在筹备</em>
        </button>
        <nav className="side-nav side-subnav" aria-label="AI 应用">
          {aiAppItems.map(({ label, icon: Icon }) => (
            <button type="button" key={label} onClick={onComingSoon}>
              <Icon size={17} />
              <span>{label}</span>
            </button>
          ))}
        </nav>
      </div>
      <div className="sidebar-spacer" />
      <section className="upgrade-card">
        <button type="button" onClick={() => onNavigate('pricing')}>
          <Sparkles size={18} /> 立即升级
        </button>
      </section>
      {currentUser ? (
        <div className="session-card">
          <span>{currentUser.email || '已登录'}</span>
          <strong className="session-credits">{Number(currentUser.credits || 0).toLocaleString('zh-CN')} 积分</strong>
          <button type="button" onClick={onLogout}>
            <LogOut size={16} /> 退出
          </button>
        </div>
      ) : (
        <button className="auth-entry" type="button" onClick={onAuthOpen}>
          登录
        </button>
      )}
      <div className="social-row">
        <a href="https://www.douyin.com/user/MS4wLjABAAAAPctRiYcwFwNx7JTqw55gxq20_jzroA_b48W1edHc7eI" target="_blank" rel="noreferrer" aria-label="Berserk AI 抖音">
          <img src="/assets/douyin-icon.svg" alt="" />
        </a>
      </div>
    </aside>
  );
}

function MobileTopbar({ onAuthOpen }) {
  return (
    <header className="mobile-topbar">
      <a className="brand" href="#" aria-label="Berserk AI">
        <img src="/assets/berserk-ai-icon.png" alt="" />
        <span>
          <strong>BERSERK AI</strong>
          <small>AI Image Studio</small>
        </span>
      </a>
      <button type="button" onClick={onAuthOpen} aria-label="打开登录">
        <Menu size={21} />
      </button>
    </header>
  );
}

function KomikoComposer({ models, feedItems, activeQuery, onQueryChange, onGenerate }) {
  const [expanded, setExpanded] = useState(false);
  const [sizeOpen, setSizeOpen] = useState(false);
  const [selectedSize, setSelectedSize] = useState('自动');
  const [selectedStyle, setSelectedStyle] = useState('艺术专业人士');
  const [selectedModel, setSelectedModel] = useState(models[0]?.id || 'gpt-image');
  const [prompt, setPrompt] = useState('');
  const [referenceCount, setReferenceCount] = useState(0);
  const [styleOpen, setStyleOpen] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchText, setSearchText] = useState('');
  const [isGenerating, setIsGenerating] = useState(false);
  const composerRef = useRef(null);
  const fileInputRef = useRef(null);
  const dynamicTags = useMemo(() => tagsFromImages(feedItems), [feedItems]);
  const currentModel = models.find((model) => model.id === selectedModel) || models[0] || defaultImageModels[0];
  const applyQuery = (value) => {
    const normalized = value.trim();
    const query = normalized === '所有帖子' || normalized === activeQuery ? '' : normalized;
    onQueryChange(query);
    setSearchText(query);
    setSearchOpen(false);
  };

  useEffect(() => {
    if (!models.some((model) => model.id === selectedModel)) {
      setSelectedModel(models[0]?.id || 'gpt-image');
    }
  }, [models, selectedModel]);

  useEffect(() => {
    const handlePointerDown = (event) => {
      if (!expanded || styleOpen) return;
      if (composerRef.current && !composerRef.current.contains(event.target)) {
        setExpanded(false);
        setSizeOpen(false);
      }
    };
    document.addEventListener('pointerdown', handlePointerDown);
    return () => document.removeEventListener('pointerdown', handlePointerDown);
  }, [expanded, styleOpen]);

  return (
    <>
      <section className="komiko-composer">
        <form
          ref={composerRef}
          className={`prompt-bar${expanded ? ' expanded' : ''}`}
          onFocus={() => setExpanded(true)}
          onClick={() => setExpanded(true)}
          onSubmit={(event) => {
            event.preventDefault();
            const cleanPrompt = prompt.trim();
            if (!cleanPrompt) {
              setExpanded(true);
              return;
            }
            setIsGenerating(true);
            Promise.resolve(onGenerate({ prompt: cleanPrompt, style: selectedStyle, size: selectedSize, modelID: selectedModel }))
              .then(() => setPrompt(''))
              .catch((error) => window.alert(getErrorMessage(error, '生成失败')))
              .finally(() => setIsGenerating(false));
          }}
        >
          {expanded ? (
            <div className="composer-tabs" aria-label="生成类型">
              <button className="active" type="button">
                <ImageIcon size={21} /> 人工智能图像
              </button>
            </div>
          ) : null}
          <Search className="prompt-search-icon" size={22} />
          {expanded ? (
            <>
              <div className="composer-toolbar">
                <span className="model-pill">{selectedStyle}</span>
                <label className="model-select-shell">
                  <img src={modelIconFor(currentModel)} alt="" />
                  <select className="model-select" value={selectedModel} onChange={(event) => setSelectedModel(event.target.value)} aria-label="选择模型">
                    {models.map((model) => (
                      <option value={model.id} key={model.id}>
                        {model.name} · {model.creditCost} 积分
                      </option>
                    ))}
                  </select>
                </label>
                <button className="style-pill" type="button" onClick={() => setStyleOpen(true)}>
                  <Palette size={15} /> 风格 <ChevronDown size={14} />
                </button>
              </div>
              <textarea value={prompt} onChange={(event) => setPrompt(event.target.value)} aria-label="图片提示词" placeholder="请用逗号分隔的短语输入提示信息。注意：此模型可能无法很好地支持原创字符。对于包含原创字符的图片，建议使用 Kokomi、Seedream 或 Gemini 等软件。" />
              <div className="composer-options">
                <div className="size-picker">
                  <button type="button" onClick={() => setSizeOpen((open) => !open)}>
                    {selectedSize} <ChevronDown size={13} />
                  </button>
                  {sizeOpen ? (
                    <div className="size-popover">
                      {generationSizes.map((size) => (
                        <button
                          className={selectedSize === size ? 'active' : ''}
                          type="button"
                          key={size}
                          onClick={() => {
                            setSelectedSize(size);
                            setSizeOpen(false);
                          }}
                        >
                          {size}
                        </button>
                      ))}
                    </div>
                  ) : null}
                </div>
                <button type="button" onClick={() => fileInputRef.current?.click()}>
                  <ImageIcon size={14} /> {referenceCount > 0 ? `参考 ${referenceCount}` : '参考'}
                </button>
                <input
                  ref={fileInputRef}
                  className="reference-input"
                  type="file"
                  accept="image/*"
                  multiple
                  webkitdirectory=""
                  directory=""
                  onChange={(event) => setReferenceCount(event.target.files?.length || 0)}
                />
              </div>
            </>
          ) : (
            <input value={prompt} onChange={(event) => setPrompt(event.target.value)} aria-label="图片提示词" placeholder="描述你想要生成的图片..." />
          )}
          <button type="submit" disabled={isGenerating}>
            {isGenerating ? '生成中' : '生成'} <Zap size={15} fill="currentColor" />
          </button>
          <img className="prompt-mascot" src="/assets/berserk-prompt-mascot-v2.png" alt="" />
        </form>
        <div className="filter-row" aria-label="筛选">
          <button type="button" onClick={() => setSearchOpen((open) => !open)}>
            <Search size={16} /> 搜索
          </button>
          <button type="button">
            <Flame size={16} /> 热门 <ChevronDown size={15} />
          </button>
          {dynamicTags.map((label, index) => (
            <button
              className={`${label === '精选' ? 'active-chip' : label === 'BerserkAIConfession' ? 'featured-chip' : ''}${activeQuery === label || (!activeQuery && label === '所有帖子') ? ' selected' : ''}`}
              type="button"
              key={label}
              onClick={() => applyQuery(label)}
            >
              {label === '精选' ? <Star size={15} /> : index > 1 ? <Hash size={14} /> : null}
              {label}
            </button>
          ))}
        </div>
      </section>
      {searchOpen ? (
        <div className="search-card-overlay" onClick={() => setSearchOpen(false)}>
          <div className="filter-search-popover" onClick={(event) => event.stopPropagation()}>
            <Search size={24} />
            <input
              value={searchText}
              onChange={(event) => setSearchText(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter') applyQuery(searchText);
              }}
              autoFocus
              placeholder="搜索帖子或生成记录"
            />
            <button type="button" aria-label="关闭搜索" onClick={() => setSearchOpen(false)}>
              <X size={20} />
            </button>
            <div className="search-card-tabs">
              <button className="active" type="button">帖子</button>
              <button type="button">生成记录</button>
            </div>
            <div className="search-card-grid">
              {feedItems.slice(0, 8).map((item) => (
                <img src={item.src} alt={item.title} key={item.id} />
              ))}
            </div>
          </div>
        </div>
      ) : null}
      {styleOpen ? (
        <StyleModal
          selectedStyle={selectedStyle}
          onSelect={(styleName) => {
            setSelectedStyle(styleName);
            setStyleOpen(false);
          }}
          onClose={() => setStyleOpen(false)}
        />
      ) : null}
    </>
  );
}

function StyleModal({ selectedStyle, onSelect, onClose }) {
  const [activeCategory, setActiveCategory] = useState('art');
  const [favoriteStyles, setFavoriteStyles] = useState(() => readStyleFavorites());
  const visibleStyles =
    activeCategory === 'favorites'
      ? stylePresets.filter((style) => favoriteStyles.includes(style.name))
      : stylePresets.filter((style) => style.category === activeCategory || (activeCategory === 'general' && ['general', 'custom'].includes(style.category)));
  const activeCategoryLabel = styleCategories.find((category) => category.id === activeCategory)?.label || '艺术';

  useEscape(onClose);

  const toggleFavorite = (styleName) => {
    setFavoriteStyles((current) => {
      const next = current.includes(styleName) ? current.filter((name) => name !== styleName) : [...current, styleName];
      window.localStorage.setItem(STYLE_FAVORITES_KEY, JSON.stringify(next));
      return next;
    });
  };

  return (
    <div className="style-overlay" role="dialog" aria-modal="true" aria-label="选择风格" onClick={onClose}>
      <div className="style-dialog" onClick={(event) => event.stopPropagation()}>
        <button className="style-close" type="button" aria-label="关闭风格选择" onClick={onClose}>
          <X size={18} />
        </button>
        <h2>选择风格</h2>
        <div className="style-layout">
          <aside className="style-sidebar">
            {styleCategories.map((category) => (
              <button className={activeCategory === category.id ? 'active' : ''} type="button" key={category.id} onClick={() => setActiveCategory(category.id)}>
                {category.id === 'favorites' ? <Star size={14} /> : null}
                {category.label}
              </button>
            ))}
          </aside>
          <section className="style-content">
            {activeCategory === 'favorites' && visibleStyles.length === 0 ? (
              <div className="style-favorites">
                <Star size={34} />
                <strong>还没有收藏</strong>
                <span>点击任意风格上的星标，可加入收藏方便快速访问</span>
              </div>
            ) : null}
            <h3>{activeCategoryLabel}</h3>
            <div className="style-grid">
              {visibleStyles.map((style) => (
                <button
                  className={selectedStyle === style.name ? 'selected' : ''}
                  type="button"
                  key={style.name}
                  onClick={() => onSelect(style.name)}
                >
                  <img src={style.image} alt={style.name} loading="lazy" />
                  <span>{style.name}</span>
                  <em className="style-badge">🌸</em>
                  <i
                    className={favoriteStyles.includes(style.name) ? 'favorited' : ''}
                    role="button"
                    tabIndex={0}
                    aria-label={favoriteStyles.includes(style.name) ? '取消收藏风格' : '收藏风格'}
                    onClick={(event) => {
                      event.stopPropagation();
                      toggleFavorite(style.name);
                    }}
                    onKeyDown={(event) => {
                      if (event.key === 'Enter' || event.key === ' ') {
                        event.preventDefault();
                        event.stopPropagation();
                        toggleFavorite(style.name);
                      }
                    }}
                  >
                    <Star size={18} fill={favoriteStyles.includes(style.name) ? 'currentColor' : 'none'} />
                  </i>
                  {selectedStyle === style.name ? <b>✓</b> : null}
                </button>
              ))}
            </div>
            {visibleStyles.length === 0 ? <p className="style-empty">当前分类暂无风格，先去其他分类看看。</p> : null}
          </section>
        </div>
      </div>
    </div>
  );
}

function MasonryFeed({ items, loading, onOpen, onLike, onFeature, onFavorite }) {
  const [visibleCount, setVisibleCount] = useState(10);
  const loaderRef = useRef(null);
  const visibleItems = useMemo(() => items.slice(0, visibleCount), [items, visibleCount]);
  const hasMore = visibleCount < items.length;

  useEffect(() => {
    setVisibleCount(10);
  }, [items]);

  useEffect(() => {
    if (!hasMore || !loaderRef.current) return undefined;
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setVisibleCount((count) => Math.min(count + 10, items.length));
        }
      },
      { rootMargin: '420px 0px' },
    );
    observer.observe(loaderRef.current);
    return () => observer.disconnect();
  }, [hasMore, visibleCount, items.length]);

  if (loading) {
    return <GallerySkeleton />;
  }

  return (
    <section className="masonry-feed" id="inspiration-feed" aria-label="图片瀑布流">
      <div className="masonry-grid">
        {visibleItems.map((item) => (
          <article className={`masonry-card${item.isFeatured || item.isPromptFeatured ? ' featured-card' : ''}`} key={item.id} onClick={() => onOpen(item)}>
            <MasonryImage item={item} />
            <span className="masonry-info">
              <span className="masonry-author-line">
                <img src={item.authorAvatarURL || '/assets/berserk-ai-icon.png'} alt="" />
                <small>{item.author}</small>
                <em>♡ {item.likeCount ?? item.likes}</em>
              </span>
            </span>
            <span className="card-actions" onClick={(event) => event.stopPropagation()}>
              <button type="button" aria-label="点赞" onClick={() => onLike(item, !item.likedByMe)}>
                {item.likedByMe ? '♥' : '♡'}
              </button>
              <button type="button" aria-label="收藏" onClick={() => onFavorite(item, !item.favoritedByMe)}>
                <Star size={15} fill={item.favoritedByMe ? 'currentColor' : 'none'} />
              </button>
            </span>
          </article>
        ))}
      </div>
      {hasMore ? (
        <button
          className="feed-loader"
          type="button"
          ref={loaderRef}
          onClick={() => setVisibleCount((count) => Math.min(count + 10, items.length))}
        >
          加载更多 <RefreshCw size={16} />
        </button>
      ) : (
        <p className="feed-end" ref={loaderRef}>已经拉到底了</p>
      )}
    </section>
  );
}

function MasonryImage({ item }) {
  const [loaded, setLoaded] = useState(false);
  const [ratio, setRatio] = useState(`${item.width || 1024} / ${item.height || 1365}`);

  return (
    <span className={`masonry-media${loaded ? ' loaded' : ''}`} style={{ aspectRatio: ratio }}>
      <img
        className="masonry-image"
        src={item.src}
        alt={`${item.author || 'Berserk AI'} 的作品`}
        loading="lazy"
        decoding="async"
        onLoad={(event) => {
          const { naturalWidth, naturalHeight } = event.currentTarget;
          if (naturalWidth > 0 && naturalHeight > 0) {
            setRatio(`${naturalWidth} / ${naturalHeight}`);
          }
          setLoaded(true);
        }}
      />
    </span>
  );
}

function GallerySkeleton() {
  const ratios = ['0.72', '1', '0.66', '1.25', '0.78', '1', '0.62', '0.86', '1.18', '0.74', '1', '0.68'];
  return (
    <section className="masonry-feed loading" id="inspiration-feed" aria-label="图片瀑布流加载中" aria-busy="true">
      <div className="masonry-grid skeleton-grid">
        {ratios.map((ratio, index) => (
          <article className="masonry-card skeleton-card" key={`${ratio}-${index}`}>
            <span className="skeleton-image" style={{ aspectRatio: ratio }} />
            <span className="skeleton-author">
              <i />
              <b />
              <em />
            </span>
          </article>
        ))}
      </div>
    </section>
  );
}

function GenerationHistory({ tasks, onRefresh }) {
  const statusText = {
    queued: '排队中',
    running: '生成中',
    succeeded: '已完成',
    failed: '失败',
  };
  return (
    <section className="generation-history">
      <header>
        <div>
          <h1>生成记录</h1>
          <p>后台异步生成中，完成后图片会自动进入首页图库。</p>
        </div>
        <button type="button" onClick={onRefresh}>
          <RefreshCw size={16} /> 刷新
        </button>
      </header>
      {tasks.length === 0 ? (
        <div className="history-empty">暂无生成记录</div>
      ) : (
        <div className="history-list">
          {tasks.map((task) => (
            <article className={`history-task ${task.status}`} key={task.id}>
              <div>
                {task.resultImage ? <img src={task.resultImage} alt="" /> : <Sparkles size={22} />}
              </div>
              <section>
                <strong>{statusText[task.status] || task.status}</strong>
                <p>{task.prompt}</p>
                {task.errorMessage ? <em>{task.errorMessage}</em> : null}
              </section>
              <span>{task.modelName || 'GPT Image'} · {task.creditsCost || 0} 积分</span>
            </article>
          ))}
        </div>
      )}
    </section>
  );
}

function ImagePreview({ item, models, currentUser, onClose, onLike, onFavorite, onGenerate, onMessage }) {
  const [panelMode, setPanelMode] = useState('detail');
  const [useReference, setUseReference] = useState(false);
  const canDownload = Number(currentUser?.totalRecharged || 0) > 0;
  useEscape(onClose);

  return (
    <div className="preview-overlay" role="dialog" aria-modal="true" aria-label={`${item.title} 预览`} onClick={onClose}>
      <div className="preview-floating-actions" onClick={(event) => event.stopPropagation()}>
        <button type="button" onClick={() => onFavorite(item, !item.favoritedByMe)}>
          <Star size={16} fill={item.favoritedByMe ? 'currentColor' : 'none'} /> 收藏
        </button>
        {canDownload ? (
          <a href={item.fullSrc || item.src} download aria-label="下载图片">
            <Download size={18} />
          </a>
        ) : (
          <button
            type="button"
            aria-label="下载图片"
            onClick={() => onMessage?.({ tone: 'warning', title: '暂不能下载', message: currentUser ? '购买过积分后即可下载原图。' : '请先登录并购买积分后再下载原图。' })}
          >
            <Download size={18} />
          </button>
        )}
        <button type="button" aria-label="关闭预览" onClick={onClose}>
          <X size={20} />
        </button>
      </div>
      <div className="preview-shell" onClick={(event) => event.stopPropagation()}>
        <div className="preview-stage">
          <img src={item.fullSrc || item.src} alt={item.title} />
        </div>
        <aside className="preview-panel">
          {panelMode === 'generate' ? (
            <PreviewGeneratePanel
              item={item}
              models={models}
              useReference={useReference}
              onBack={() => setPanelMode('detail')}
              onGenerate={onGenerate}
            />
          ) : (
            <>
              <div className="preview-author">
                <img src={item.authorAvatarURL || '/assets/berserk-ai-icon.png'} alt="" />
                <span>
                  <strong>{item.author}</strong>
                  <small>@{String(item.author || 'BerserkAI').replace(/\s+/g, '')}</small>
                </span>
              </div>
              <div className="preview-stats">
                <button type="button" onClick={() => onLike(item, !item.likedByMe)}>{item.likedByMe ? '♥' : '♡'} {item.likeCount ?? item.likes}</button>
                <span><BarChart3 size={16} /> {formatViews((item.likeCount || item.likes || 0) * 21 + 420)}</span>
                <span>{relativeTime(item.createdAt)}</span>
              </div>
              <div className={`preview-prompt${item.isPromptFeatured ? ' prompt-featured' : ''}`}>
                <p>{item.promptZh}</p>
              </div>
              <div className="preview-tools">
                <span><img src={modelIconFor(item)} alt="" /> {item.model}</span>
                <button type="button" onClick={() => navigator.clipboard?.writeText(item.promptZh || '')}><Copy size={16} /> 复制</button>
              </div>
              <div className="preview-bottom-actions">
                <button className="preview-action" type="button" onClick={() => {
                  setUseReference(false);
                  setPanelMode('generate');
                }}>
                  <Sparkles size={18} /> 使用提示词
                </button>
                <button className="preview-action" type="button" onClick={() => {
                  setUseReference(true);
                  setPanelMode('generate');
                }}>
                  <ImageIcon size={18} /> 用作参考图
                </button>
              </div>
            </>
          )}
        </aside>
      </div>
    </div>
  );
}

function PreviewGeneratePanel({ item, models, useReference, onBack, onGenerate }) {
  const [promptText, setPromptText] = useState(item.promptZh || '');
  const [selectedModel, setSelectedModel] = useState(item.modelID || models[0]?.id || 'gpt-image');
  const [selectedSize, setSelectedSize] = useState('自动');
  const [quality, setQuality] = useState('standard');
  const [quantity, setQuantity] = useState(1);
  const [resolution2K, setResolution2K] = useState(false);
  const [sizeMenuOpen, setSizeMenuOpen] = useState(false);
  const [negativePrompt, setNegativePrompt] = useState('');
  const [localRefs, setLocalRefs] = useState([]);
  const [busy, setBusy] = useState(false);
  const fileInputRef = useRef(null);
  const currentModel = models.find((model) => model.id === selectedModel) || models[0] || defaultImageModels[0];
  const creditCost = (currentModel?.creditCost || 5) * quantity;
  const referenceImages = [
    ...(useReference ? [{ src: item.fullSrc || item.src, preview: item.src }] : []),
    ...localRefs,
  ].slice(0, 5);

  useEffect(() => {
    if (!models.some((model) => model.id === selectedModel)) {
      setSelectedModel(models[0]?.id || 'gpt-image');
    }
  }, [models, selectedModel]);

  const submit = () => {
    const cleanPrompt = promptText.trim();
    if (!cleanPrompt) return;
    setBusy(true);
    Promise.resolve(onGenerate({
      prompt: cleanPrompt,
      style: item.style,
      size: selectedSize,
      modelID: selectedModel,
      images: referenceImages.map((image) => image.src),
      quality,
      n: quantity,
      negativePrompt,
      resolution: resolution2K ? '2k' : 'auto',
    }))
      .catch((error) => window.alert(getErrorMessage(error, '生成失败')))
      .finally(() => setBusy(false));
  };

  const improvePrompt = () => {
    setPromptText((value) => {
      const clean = value.trim();
      if (!clean) return clean;
      if (clean.includes('高质量细节')) return clean;
      return `${clean}\n\n高质量细节，清晰主体，商业级构图，精致光影，画面干净。`;
    });
  };

  const desaturatePrompt = () => {
    setPromptText((value) => {
      const clean = value.trim();
      if (!clean) return clean;
      if (clean.includes('低饱和')) return clean;
      return `${clean}\n\n低饱和配色，柔和色阶，避免过度鲜艳。`;
    });
  };

  const handleReferenceFiles = (event) => {
    const files = Array.from(event.target.files || []).slice(0, Math.max(0, 5 - referenceImages.length));
    files.forEach((file) => {
      const reader = new FileReader();
      reader.onload = () => {
        if (typeof reader.result === 'string') {
          setLocalRefs((items) => [...items, { src: reader.result, preview: reader.result }].slice(0, 5));
        }
      };
      reader.readAsDataURL(file);
    });
    event.target.value = '';
  };

  return (
    <div className="preview-generate-panel">
      <header>
        <strong>生成</strong>
        <button type="button" aria-label="返回详情" onClick={onBack}>
          <LayoutTemplate size={17} />
        </button>
      </header>
      <div className="reference-strip">
        <span><ImageIcon size={15} /> {referenceImages.length}/5</span>
        {referenceImages[0] ? <img src={referenceImages[0].preview} alt="" /> : null}
        <button type="button" onClick={() => fileInputRef.current?.click()} disabled={referenceImages.length >= 5}>
          <Plus size={20} />
        </button>
        <input ref={fileInputRef} type="file" accept="image/*" multiple onChange={handleReferenceFiles} />
      </div>
      <label className="generate-prompt-box">
        <span>
          画面描述 <em>使用</em> <b>引用参考图</b>
          <button type="button" aria-label="复制提示词" onClick={() => navigator.clipboard?.writeText(promptText)}>
            <Copy size={14} />
          </button>
        </span>
        <textarea value={promptText} onChange={(event) => setPromptText(event.target.value)} />
        <div>
          <button type="button" onClick={improvePrompt}>AI 帮改</button>
          <button type="button" onClick={desaturatePrompt}>消色</button>
          <button type="button" onClick={submit}>⌘ + ↵</button>
        </div>
      </label>
      <div className="generate-setting-row">
        <button type="button" onClick={() => setQuantity((value) => Math.max(1, value - 1))}>−</button>
        <button type="button" onClick={() => setQuantity((value) => (value >= 4 ? 1 : value + 1))}>{quantity}/4</button>
        <div className="size-select">
          <button type="button" onClick={() => setSizeMenuOpen((value) => !value)}><LayoutTemplate size={14} /> {selectedSize}</button>
          {sizeMenuOpen ? (
            <div className="size-menu">
              {generationSizes.map((size) => (
                <button
                  className={selectedSize === size ? 'active' : ''}
                  type="button"
                  key={size}
                  onClick={() => {
                    setSelectedSize(size);
                    setSizeMenuOpen(false);
                  }}
                >
                  {size}
                </button>
              ))}
            </div>
          ) : null}
        </div>
        <button type="button" className={resolution2K ? 'active' : ''} onClick={() => setResolution2K((value) => !value)}>2K</button>
      </div>
      <div className="quality-group" aria-label="选择生成质量">
        {[
          ['standard', '标准'],
          ['medium', 'Medium'],
          ['high', 'High'],
        ].map(([value, label]) => (
          <button className={quality === value ? 'active' : ''} type="button" key={value} onClick={() => setQuality(value)}>
            {label}
          </button>
        ))}
      </div>
      <label className="generate-model-select">
        <img src={modelIconFor(currentModel)} alt="" />
        <select value={selectedModel} onChange={(event) => setSelectedModel(event.target.value)}>
          {models.map((model) => (
            <option value={model.id} key={model.id}>{model.name} · {model.creditCost} 积分</option>
          ))}
        </select>
      </label>
      <button className="generate-submit" type="button" onClick={submit} disabled={busy || !promptText.trim()}>
        {busy ? '生成中' : `生成图片 ✨ ${creditCost}`}
      </button>
    </div>
  );
}

function AppModal({ title, message, tone = 'info', onClose }) {
  useEscape(onClose);
  return (
    <div className="app-modal-backdrop" onMouseDown={onClose}>
      <div className={`app-modal ${tone}`} role="dialog" aria-modal="true" onMouseDown={(event) => event.stopPropagation()}>
        <img className="app-modal-decoration" src="/assets/modal-crystal-decoration.png" alt="" />
        <button className="app-modal-close" type="button" aria-label="关闭" onClick={onClose}>
          <X size={18} />
        </button>
        <span className="app-modal-icon">
          <Sparkles size={26} />
        </span>
        <h2>{title}</h2>
        <p>{message}</p>
        <button className="app-modal-primary" type="button" onClick={onClose}>知道了</button>
      </div>
    </div>
  );
}

function AuthModal({ onClose, onSuccess }) {
  const [mode, setMode] = useState('login');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [code, setCode] = useState('');
  const [statusMessage, setStatusMessage] = useState('');
  const [error, setError] = useState('');
  const [isSendingCode, setIsSendingCode] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [codeSent, setCodeSent] = useState(false);
  const cleanEmail = email.trim().toLowerCase();
  const isRegister = mode === 'register';
  const isCodeLogin = mode === 'login-code';
  const needsCode = isRegister || isCodeLogin;
  const title = isRegister ? '注册 BerserkAI' : '欢迎来到 BerserkAI';
  const subtitle = isRegister ? '注册后即可保存你的创作记录' : '登录后同步保存你的灵感与作品';
  const canRequestCode = Boolean(cleanEmail) && (!isRegister || password.trim().length >= 8);
  const canSubmit =
    cleanEmail &&
    (isCodeLogin ? code.trim().length === 6 : isRegister ? password.trim().length >= 8 && code.trim().length === 6 : Boolean(password.trim()));

  useEscape(onClose);

  useEffect(() => {
    if (countdown <= 0) return undefined;
    const timer = window.setTimeout(() => {
      setCountdown((value) => Math.max(0, value - 1));
    }, 1000);
    return () => window.clearTimeout(timer);
  }, [countdown]);

  const resetTransientState = () => {
    setCodeSent(false);
    setCode('');
    setStatusMessage('');
    setError('');
    setCountdown(0);
  };

  const switchMode = (nextMode) => {
    setMode(nextMode);
    setPassword('');
    resetTransientState();
  };

  const handleRequestCode = async () => {
    setError('');
    setStatusMessage('');
    setIsSendingCode(true);
    try {
      const result = await requestEmailCode({ email: cleanEmail, mode: isRegister ? 'register' : 'login' });
      setCountdown(result?.expiresIn || 90);
      setCodeSent(true);
      setStatusMessage(`验证码已发送到 ${cleanEmail}。`);
    } catch (requestError) {
      setError(getErrorMessage(requestError, '验证码发送失败'));
    } finally {
      setIsSendingCode(false);
    }
  };

  const handleSubmit = async (event) => {
    event.preventDefault();
    setError('');
    setStatusMessage('');
    setIsSubmitting(true);
    try {
      const session = isRegister
        ? await registerWithEmail({ email: cleanEmail, password, code })
        : isCodeLogin
          ? await loginWithEmailCode({ email: cleanEmail, code })
          : await loginWithPassword({ email: cleanEmail, password });
      onSuccess(session);
    } catch (submitError) {
      setError(getErrorMessage(submitError, isRegister ? '注册失败' : '登录失败'));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="auth-overlay" role="dialog" aria-modal="true" aria-label={isRegister ? '邮箱注册' : '邮箱登录'} onClick={onClose}>
      <div className="auth-card" onClick={(event) => event.stopPropagation()}>
        <button className="modal-close auth-close" aria-label="关闭登录" onClick={onClose}>
          <X size={18} />
        </button>
        <div className="auth-panel">
          <div className="auth-mark">
            <img src="/assets/berserk-ai-icon.png" alt="" />
            <span>BERSERK AI</span>
          </div>
          <h2>{title}</h2>
          <p>{subtitle}</p>
          <form className="auth-form" onSubmit={handleSubmit}>
            <input
              id="auth-email"
              type="email"
              value={email}
              onChange={(event) => {
                setEmail(event.target.value);
                resetTransientState();
              }}
              placeholder="name@example.com"
              autoComplete="email"
              required
            />
            {!isCodeLogin ? (
              <input
                type="password"
                value={password}
                onChange={(event) => {
                  setPassword(event.target.value);
                  if (isRegister) resetTransientState();
                }}
                placeholder={isRegister ? '设置登录密码（至少 8 位）' : '登录密码'}
                autoComplete={isRegister ? 'new-password' : 'current-password'}
                required
              />
            ) : null}
            {needsCode ? (
              <div className="auth-code-line">
                <input
                  id="auth-code"
                  type="text"
                  inputMode="numeric"
                  value={code}
                  onChange={(event) => setCode(event.target.value.replace(/\D/g, '').slice(0, 6))}
                  placeholder="邮箱验证码"
                  autoComplete="one-time-code"
                  required
                />
                <button className="auth-code-button" type="button" onClick={handleRequestCode} disabled={!canRequestCode || isSendingCode || countdown > 0}>
                  {isSendingCode ? '发送中' : countdown > 0 ? `${countdown}s` : codeSent ? '重发' : '获取验证码'}
                </button>
              </div>
            ) : null}
            {mode === 'login' ? (
              <button className="auth-switch" type="button" onClick={() => switchMode('login-code')}>
                使用邮箱验证码登录
              </button>
            ) : null}
            {isCodeLogin ? (
              <button className="auth-switch" type="button" onClick={() => switchMode('login')}>
                使用密码登录
              </button>
            ) : null}
            {error ? (
              <div className="auth-note error" role="alert">
                {error}
              </div>
            ) : null}
            {statusMessage ? <div className="auth-note">{statusMessage}</div> : null}
            <button className="auth-submit" type="submit" disabled={!canSubmit || isSubmitting}>
              {isSubmitting ? <LoaderCircle size={18} /> : null}
              {isSubmitting ? (isRegister ? '注册中' : '登录中') : isRegister ? '使用邮箱注册' : isCodeLogin ? '验证码登录' : '登录'}
            </button>
            {isRegister ? (
              <button className="auth-help" type="button" onClick={() => switchMode('login')}>
                已有账号？去登录
              </button>
            ) : (
              <button className="auth-help" type="button" onClick={() => switchMode('register')}>
                注册账号
              </button>
            )}
          </form>
          <p className="auth-terms">登录或注册即表示您同意我们的服务条款和隐私政策。</p>
        </div>
      </div>
    </div>
  );
}

function PricingPage({ packages, authSession, onAuthOpen, onUserChange, onBack }) {
  const [redeemCardNo, setRedeemCardNo] = useState('');
  const [redeemPassword, setRedeemPassword] = useState('');
  const [redeemMessage, setRedeemMessage] = useState('');
  const [busyPackage, setBusyPackage] = useState('');
  const [openFaq, setOpenFaq] = useState('');

  const handlePurchase = async (pkg) => {
    if (!authSession?.token) {
      onAuthOpen();
      return;
    }
    setBusyPackage(pkg.id);
    setRedeemMessage('');
    try {
      const payload = await authPostJSON('/api/v1/credits/purchase', authSession.token, { packageID: pkg.id }, '创建订单失败');
      if (payload?.paymentURL) window.open(payload.paymentURL, '_blank', 'noopener,noreferrer');
      if (payload?.user) onUserChange(payload.user);
      setRedeemMessage(payload?.paymentURL ? '已打开卡密购买页面，支付后回到这里兑换卡密。' : '积分已到账。');
    } catch (error) {
      setRedeemMessage(getErrorMessage(error, '购买失败'));
    } finally {
      setBusyPackage('');
    }
  };

  const handleRedeem = async (event) => {
    event.preventDefault();
    if (!authSession?.token) {
      onAuthOpen();
      return;
    }
    setRedeemMessage('');
    try {
      const payload = await authPostJSON('/api/v1/credits/redeem', authSession.token, { cardNo: redeemCardNo.trim(), password: redeemPassword.trim() }, '卡密兑换失败');
      if (payload?.user) onUserChange(payload.user);
      setRedeemCardNo('');
      setRedeemPassword('');
      setRedeemMessage(`兑换成功，已到账 ${payload?.credits || 0} 积分。`);
    } catch (error) {
      setRedeemMessage(getErrorMessage(error, '兑换失败'));
    }
  };

  return (
    <section className="pricing-page">
      <nav className="pricing-nav" aria-label="订阅导航">
        <button type="button" onClick={onBack}>
          BERSERK AI
        </button>
        <div>
          <a href="#pricing-plans">积分套餐</a>
          <a href="#pricing-faq">常见问题</a>
          <button type="button" onClick={onBack}>
            立即创作
          </button>
        </div>
      </nav>
      <header className="pricing-hero">
        <h1>购买 Berserk AI 积分</h1>
        <button type="button">一次性积分包</button>
        <p>不同模型按配置消耗积分。通过卡密平台购买后，回到本页输入卡密兑换积分。</p>
        <div className="payment-coming">微信支付、支付宝支付即将上线</div>
      </header>
      <form className="redeem-panel" onSubmit={handleRedeem}>
        <div>
          <strong>卡密兑换</strong>
          <span>输入卡号和密码，在这里兑换到当前账号。</span>
        </div>
        <input value={redeemCardNo} onChange={(event) => setRedeemCardNo(event.target.value)} placeholder="卡号" />
        <input value={redeemPassword} onChange={(event) => setRedeemPassword(event.target.value)} placeholder="密码" />
        <button type="submit">兑换积分</button>
      </form>
      {redeemMessage ? <p className="redeem-message">{redeemMessage}</p> : null}
      <div className="pricing-grid" id="pricing-plans">
        {packages.map((pkg) => (
          <article className={`pricing-card ${pkg.tone}${pkg.popular ? ' popular' : ''}`} key={pkg.id}>
            {pkg.popular ? <em>推荐</em> : null}
            <img className="pricing-icon" src={pkg.icon} alt="" />
            <h2>{pkg.name}</h2>
            <div className="price-line">
              <strong>{pkg.price}</strong>
            </div>
            <div className="zap-amount">
              <Zap size={22} fill="currentColor" /> {pkg.credits}
            </div>
            <ul>
              {pkg.features.map((feature) => (
                <li key={feature}>✓ {feature}</li>
              ))}
            </ul>
            <button type="button" onClick={() => handlePurchase(pkg)} disabled={busyPackage === pkg.id}>
              {busyPackage === pkg.id ? '处理中' : pkg.paymentURL ? '去购买卡密' : '购买积分'}
            </button>
          </article>
        ))}
      </div>
      <section className="pricing-proof">
        <h2>按需购买，不绑定订阅</h2>
        <p>积分包适合灵活创作：买多少用多少，生成失败会按后端状态退还积分。</p>
        <div>
          <span>5 积分 / 次生成</span>
          <span>立即到账</span>
          <span>支持多次购买</span>
        </div>
      </section>
      <section className="pricing-faq" id="pricing-faq">
        <h2>常见问题</h2>
        {pricingFaqs.map((item) => (
          <div className={`faq-item${openFaq === item.question ? ' open' : ''}`} key={item.question}>
            <button
              type="button"
              aria-expanded={openFaq === item.question}
              onClick={() => setOpenFaq((current) => (current === item.question ? '' : item.question))}
            >
              {item.question} <ChevronDown size={18} />
            </button>
            {openFaq === item.question ? <p>{item.answer}</p> : null}
          </div>
        ))}
      </section>
    </section>
  );
}

function Footer() {
  return (
    <footer className="site-footer">
      <div>
        <a className="footer-brand" href="#">
          <img src="/assets/berserk-ai-icon.png" alt="" /> Berserk AI
        </a>
        <p>Copyright © 2026 保留所有权利。</p>
      </div>
      <div>
        <h3>AI 模型</h3>
        <a href="#">Gemini</a>
        <a href="#">NoobAI XL</a>
      </div>
      <div>
        <h3>插画工具</h3>
        <a href="#">角色创建器</a>
        <a href="#">AI 艺术生成器</a>
      </div>
      <div>
        <h3>动画工具</h3>
        <a href="#">AI 动画制作工具</a>
        <a href="#">AI 动态人像</a>
      </div>
      <div>
        <h3>漫画工具</h3>
        <a href="#">漫画画布</a>
        <a href="#">AI 漫画生成器</a>
      </div>
      <div>
        <h3>了解更多</h3>
        <a href="#">价格</a>
        <a href="#">博客</a>
      </div>
    </footer>
  );
}

function ProfileModal({ session, onClose, onAuthOpen, onUserChange }) {
  const user = session?.user;
  const [displayName, setDisplayName] = useState(user?.displayName || '');
  const [avatarURL, setAvatarURL] = useState(user?.avatarURL || '');
  const [signature, setSignature] = useState(user?.signature || '');
  const [gender, setGender] = useState(user?.gender || '');
  const [message, setMessage] = useState('');
  const [saving, setSaving] = useState(false);

  useEscape(onClose);

  const handleAvatarFile = (event) => {
    const file = event.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => setAvatarURL(String(reader.result || ''));
    reader.readAsDataURL(file);
  };

  const handleSubmit = async (event) => {
    event.preventDefault();
    if (!session?.token) {
      onAuthOpen();
      return;
    }
    setSaving(true);
    setMessage('');
    try {
      const updated = await authPatchJSON('/api/v1/me/profile', session.token, { displayName, avatarURL, signature, gender }, '保存资料失败');
      onUserChange(updated);
      setMessage('资料已保存。');
    } catch (error) {
      setMessage(getErrorMessage(error, '保存失败'));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="profile-overlay" role="dialog" aria-modal="true" aria-label="用户资料" onClick={onClose}>
      <form className="profile-card" onSubmit={handleSubmit} onClick={(event) => event.stopPropagation()}>
        <button className="modal-close profile-close" type="button" aria-label="关闭资料卡" onClick={onClose}>
          <X size={18} />
        </button>
        <div className="profile-hero">
          <label className="avatar-picker">
            {avatarURL ? <img src={avatarURL} alt="" /> : <span>{(user?.email || 'B').slice(0, 1).toUpperCase()}</span>}
            <input type="file" accept="image/*" onChange={handleAvatarFile} />
          </label>
          <div>
            <strong>{user?.email || '未登录用户'}</strong>
            <span>{user?.credits ?? 0} Zaps</span>
          </div>
        </div>
        <label>
          昵称
          <input value={displayName} onChange={(event) => setDisplayName(event.target.value)} placeholder="给自己起个创作者名字" />
        </label>
        <label>
          性别
          <select value={gender} onChange={(event) => setGender(event.target.value)}>
            <option value="">不展示</option>
            <option value="female">女</option>
            <option value="male">男</option>
            <option value="nonbinary">非二元</option>
          </select>
        </label>
        <label>
          个性签名
          <textarea value={signature} onChange={(event) => setSignature(event.target.value)} placeholder="写一句会显示在资料卡上的创作宣言" />
        </label>
        {message ? <p>{message}</p> : null}
        <button type="submit" disabled={saving}>
          {saving ? '保存中' : '保存资料'}
        </button>
      </form>
    </div>
  );
}

function useEscape(onClose) {
  useEffect(() => {
    const handleKeydown = (event) => {
      if (event.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', handleKeydown);
    return () => window.removeEventListener('keydown', handleKeydown);
  }, [onClose]);
}

function readStoredAuthSession() {
  try {
    return JSON.parse(window.localStorage.getItem(AUTH_STORAGE_KEY) || 'null');
  } catch {
    return null;
  }
}

function readStyleFavorites() {
  try {
    const value = JSON.parse(window.localStorage.getItem(STYLE_FAVORITES_KEY) || '[]');
    return Array.isArray(value) ? value : [];
  } catch {
    return [];
  }
}

async function requestEmailCode({ email, mode }) {
  return postJSON('/api/v1/auth/email/code', { email: email.trim().toLowerCase(), mode, appID: AUTH_APP_ID }, '验证码发送失败');
}

async function loginWithEmailCode({ email, code }) {
  return postJSON('/api/v1/auth/email/login', { email: email.trim().toLowerCase(), code: code.trim(), appID: AUTH_APP_ID }, '登录失败');
}

async function loginWithPassword({ email, password }) {
  return postJSON('/api/v1/auth/email/login', { email: email.trim().toLowerCase(), password, appID: AUTH_APP_ID }, '登录失败');
}

async function registerWithEmail({ email, password, code }) {
  return postJSON(
    '/api/v1/auth/email/register',
    { email: email.trim().toLowerCase(), password, code: code.trim(), appID: AUTH_APP_ID },
    '注册失败',
  );
}

function normalizeCreditPackage(pkg) {
  const credits = Number(pkg.credits || 0);
  const price = `¥${Math.round(Number(pkg.amountCents || 0) / 100)}`;
  const fallbackIcons = {
    credits_trial: '/pricing-icons/package-trial.png',
    credits_100: '/pricing-icons/package-100.png',
    credits_500: '/pricing-icons/package-500.png',
    credits_1000: '/pricing-icons/package-1000.png',
  };
  return {
    id: pkg.id,
    name: pkg.name,
    price,
    credits: `${credits.toLocaleString('zh-CN')} 积分`,
    icon: pkg.icon || fallbackIcons[pkg.id] || `/pricing-icons/${pkg.id}.png`,
    paymentURL: pkg.paymentURL || '',
    tone: credits >= 1000 ? 'gold' : credits >= 500 ? 'purple' : 'blue',
    popular: credits === 500,
    features: [`可兑换 ${credits.toLocaleString('zh-CN')} 积分`, `约可生成 ${Math.max(1, Math.floor(credits / 5))} 张基础模型图片`, '支持卡密兑换到账', '积分长期保留'],
  };
}

function sizeToBackendSize(size) {
  const map = {
    '自动': '1024x1365',
    '1:1': '1024x1024',
    '3:4': '1024x1365',
    '4:5': '1024x1280',
    '4:3': '1365x1024',
    '9:16': '1024x1792',
    '16:9': '1792x1024',
    '21:9': '1792x768',
    '2:3': '1024x1536',
    '5:4': '1280x1024',
    '3:2': '1536x1024',
  };
  return map[size] || '1024x1365';
}

function relativeTime(value) {
  if (!value) return '刚刚';
  const then = new Date(value).getTime();
  if (!Number.isFinite(then)) return '刚刚';
  const seconds = Math.max(1, Math.floor((Date.now() - then) / 1000));
  if (seconds < 60) return `${seconds} 秒前`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes} 分钟前`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} 小时前`;
  return `${Math.floor(hours / 24)} 天前`;
}

function formatViews(value) {
  const count = Number(value || 0);
  if (count >= 10000) return `${(count / 10000).toFixed(1)}万`;
  if (count >= 1000) return `${(count / 1000).toFixed(1)}K`;
  return String(Math.max(0, Math.round(count)));
}

async function getJSON(path, token = '') {
  const headers = token ? { Authorization: `Bearer ${token}` } : undefined;
  const response = await fetch(`${API_BASE_URL}${path}`, { headers });
  const payload = await response.json().catch(() => null);
  if (!response.ok) throw new Error(localizeError(payload?.message, `请求失败（${response.status}）`));
  return payload;
}

async function preloadGalleryImages(urls) {
  const uniqueURLs = Array.from(new Set(urls.filter(Boolean)));
  if (uniqueURLs.length === 0) return;
  const timeout = new Promise((resolve) => window.setTimeout(resolve, 1800));
  const load = Promise.allSettled(
    uniqueURLs.map((url) => new Promise((resolve) => {
      const image = new Image();
      image.decoding = 'async';
      image.onload = () => {
        if (image.decode) {
          image.decode().then(resolve).catch(resolve);
          return;
        }
        resolve();
      };
      image.onerror = resolve;
      image.src = url;
    })),
  );
  await Promise.race([load, timeout]);
}

function authPostJSON(path, token, body, fallbackMessage) {
  return authedJSON(path, token, 'POST', body, fallbackMessage);
}

function authPatchJSON(path, token, body, fallbackMessage) {
  return authedJSON(path, token, 'PATCH', body, fallbackMessage);
}

async function authedJSON(path, token, method, body, fallbackMessage) {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(body),
  });
  const payload = await response.json().catch(() => null);
  if (!response.ok) {
    throw new Error(localizeError(payload?.message, `${fallbackMessage}（${response.status}）`));
  }
  return payload;
}

async function postJSON(path, body, fallbackMessage) {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(body),
  });
  const payload = await response.json().catch(() => null);
  if (!response.ok) {
    throw new Error(localizeError(payload?.message, `${fallbackMessage}（${response.status}）`));
  }
  return payload;
}

function getErrorMessage(error, fallbackMessage) {
  return error instanceof Error ? localizeError(error.message, fallbackMessage) : fallbackMessage;
}

function localizeError(message, fallbackMessage = '操作失败，请稍后重试') {
  const text = String(message || '').trim();
  const dictionary = {
    'Failed to fetch': '无法连接到后端服务，请确认本地 8080 服务已启动',
    'NetworkError when attempting to fetch resource.': '网络请求失败，请稍后重试',
    'smtp is not configured': '邮箱服务未配置',
    'email already registered': '该邮箱已经注册',
    'email is not registered': '该邮箱尚未注册',
    'invalid email': '邮箱地址不正确',
    'invalid or expired code': '验证码无效或已过期',
    '验证码无效或已过期，请重新获取': '验证码无效或已过期，请重新获取',
    'invalid or expired verification': '邮箱验证已过期，请重新获取验证码',
    'password must be at least 8 characters': '密码至少需要 8 位',
    'invalid email or password': '邮箱或密码不正确',
    'credits are not enough': '积分不足，请先充值',
    '已有图片正在生成，请完成后再提交新的任务': '已有图片正在生成，请完成后再提交新的任务',
    'invalid image task payload': '生图请求格式不正确',
    'prompt is required': '请输入提示词',
    'invalid image model': '请选择可用的生图模型',
    'create image task failed': '创建生成任务失败，请稍后重试',
  };
  if (dictionary[text]) return dictionary[text];
  if (/^[\x00-\x7F\s.,:;!?()'"/_-]+$/.test(text)) return fallbackMessage;
  return text || fallbackMessage;
}

export default App;
