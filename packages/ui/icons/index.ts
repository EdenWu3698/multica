import type { ForwardRefExoticComponent, RefAttributes, SVGProps } from "react";
import {
  Activity as LucideActivity,
  AlarmClock as LucideAlarmClock,
  AlertCircle as LucideAlertCircle,
  AlertTriangle as LucideAlertTriangle,
  AppWindow as LucideAppWindow,
  Archive as LucideArchive,
  ArchiveRestore as LucideArchiveRestore,
  ArrowBigUp as LucideArrowBigUp,
  ArrowDown as LucideArrowDown,
  ArrowDownNarrowWide as LucideArrowDownNarrowWide,
  ArrowDownToLine as LucideArrowDownToLine,
  ArrowLeft as LucideArrowLeft,
  ArrowLeftRight as LucideArrowLeftRight,
  ArrowRight as LucideArrowRight,
  ArrowRightToLine as LucideArrowRightToLine,
  ArrowUp as LucideArrowUp,
  ArrowUpCircle as LucideArrowUpCircle,
  ArrowUpDown as LucideArrowUpDown,
  ArrowUpNarrowWide as LucideArrowUpNarrowWide,
  ArrowUpRight as LucideArrowUpRight,
  AtSign as LucideAtSign,
  Ban as LucideBan,
  BarChart3 as LucideBarChart3,
  Bell as LucideBell,
  Blocks as LucideBlocks,
  Bold as LucideBold,
  BookCheck as LucideBookCheck,
  BookOpen as LucideBookOpen,
  BookOpenText as LucideBookOpenText,
  Bookmark as LucideBookmark,
  Bot as LucideBot,
  BotOff as LucideBotOff,
  Box as LucideBox,
  Brain as LucideBrain,
  Briefcase as LucideBriefcase,
  BriefcaseBusiness as LucideBriefcaseBusiness,
  Bug as LucideBug,
  Building2 as LucideBuilding2,
  Calendar as LucideCalendar,
  CalendarClock as LucideCalendarClock,
  CalendarDays as LucideCalendarDays,
  Camera as LucideCamera,
  ChartGantt as LucideChartGantt,
  Check as LucideCheck,
  CheckCheck as LucideCheckCheck,
  CheckCircle2 as LucideCheckCircle2,
  CheckIcon as LucideCheckIcon,
  ChevronDown as LucideChevronDown,
  ChevronDownIcon as LucideChevronDownIcon,
  ChevronLeft as LucideChevronLeft,
  ChevronLeftIcon as LucideChevronLeftIcon,
  ChevronRight as LucideChevronRight,
  ChevronRightIcon as LucideChevronRightIcon,
  ChevronUp as LucideChevronUp,
  ChevronUpIcon as LucideChevronUpIcon,
  ChevronsUpDown as LucideChevronsUpDown,
  Circle as LucideCircle,
  CircleAlert as LucideCircleAlert,
  CircleCheck as LucideCircleCheck,
  CircleCheckIcon as LucideCircleCheckIcon,
  CircleDashed as LucideCircleDashed,
  CircleDot as LucideCircleDot,
  CircleEqual as LucideCircleEqual,
  CircleHelp as LucideCircleHelp,
  CircleOff as LucideCircleOff,
  CirclePause as LucideCirclePause,
  CircleSlash as LucideCircleSlash,
  CircleUser as LucideCircleUser,
  Clock as LucideClock,
  Clock3 as LucideClock3,
  Cloud as LucideCloud,
  Code as LucideCode,
  Code2 as LucideCode2,
  CodeXml as LucideCodeXml,
  Coins as LucideCoins,
  Columns2 as LucideColumns2,
  Columns3 as LucideColumns3,
  Command as LucideCommand,
  Compass as LucideCompass,
  Component as LucideComponent,
  Copy as LucideCopy,
  CornerDownLeft as LucideCornerDownLeft,
  CornerDownRight as LucideCornerDownRight,
  CornerLeftUp as LucideCornerLeftUp,
  CornerUpLeft as LucideCornerUpLeft,
  Cpu as LucideCpu,
  CreditCard as LucideCreditCard,
  Crown as LucideCrown,
  Database as LucideDatabase,
  Delete as LucideDelete,
  Dices as LucideDices,
  Download as LucideDownload,
  Ellipsis as LucideEllipsis,
  ExternalLink as LucideExternalLink,
  Eye as LucideEye,
  EyeOff as LucideEyeOff,
  FastForward as LucideFastForward,
  File as LucideFile,
  FileArchive as LucideFileArchive,
  FileAudio as LucideFileAudio,
  FileCode as LucideFileCode,
  FileEdit as LucideFileEdit,
  FileImage as LucideFileImage,
  FilePen as LucideFilePen,
  FilePlus as LucideFilePlus,
  FilePlus2 as LucideFilePlus2,
  FileQuestion as LucideFileQuestion,
  FileSearch as LucideFileSearch,
  FileSpreadsheet as LucideFileSpreadsheet,
  FileText as LucideFileText,
  FileVideo as LucideFileVideo,
  Filter as LucideFilter,
  FilterX as LucideFilterX,
  Flag as LucideFlag,
  Folder as LucideFolder,
  FolderGit as LucideFolderGit,
  FolderGit2 as LucideFolderGit2,
  FolderKanban as LucideFolderKanban,
  FolderMinus as LucideFolderMinus,
  FolderOpen as LucideFolderOpen,
  Frame as LucideFrame,
  Gauge as LucideGauge,
  GitBranch as LucideGitBranch,
  GitMerge as LucideGitMerge,
  GitPullRequest as LucideGitPullRequest,
  GitPullRequestArrow as LucideGitPullRequestArrow,
  GitPullRequestClosed as LucideGitPullRequestClosed,
  GitPullRequestDraft as LucideGitPullRequestDraft,
  Globe as LucideGlobe,
  Globe2 as LucideGlobe2,
  GraduationCap as LucideGraduationCap,
  GripVertical as LucideGripVertical,
  HardDrive as LucideHardDrive,
  Hash as LucideHash,
  Heading1 as LucideHeading1,
  Heading2 as LucideHeading2,
  Heading3 as LucideHeading3,
  Heart as LucideHeart,
  HelpCircle as LucideHelpCircle,
  Highlighter as LucideHighlighter,
  History as LucideHistory,
  Hourglass as LucideHourglass,
  Image as LucideImage,
  ImageIcon as LucideImageIcon,
  ImagePlus as LucideImagePlus,
  Inbox as LucideInbox,
  Info as LucideInfo,
  InfoIcon as LucideInfoIcon,
  Italic as LucideItalic,
  KeyRound as LucideKeyRound,
  Keyboard as LucideKeyboard,
  Laptop as LucideLaptop,
  Layers as LucideLayers,
  Layers3 as LucideLayers3,
  LayoutDashboard as LucideLayoutDashboard,
  LayoutGrid as LucideLayoutGrid,
  LayoutTemplate as LucideLayoutTemplate,
  Lightbulb as LucideLightbulb,
  Link as LucideLink,
  Link2 as LucideLink2,
  List as LucideList,
  ListChecks as LucideListChecks,
  ListChevronsDownUp as LucideListChevronsDownUp,
  ListChevronsUpDown as LucideListChevronsUpDown,
  ListEnd as LucideListEnd,
  ListOrdered as LucideListOrdered,
  ListPlus as LucideListPlus,
  ListTodo as LucideListTodo,
  ListTree as LucideListTree,
  ListX as LucideListX,
  Loader2 as LucideLoader2,
  Loader2Icon as LucideLoader2Icon,
  LoaderCircle as LucideLoaderCircle,
  LocateFixed as LucideLocateFixed,
  Lock as LucideLock,
  LockKeyhole as LucideLockKeyhole,
  LogIn as LucideLogIn,
  LogOut as LucideLogOut,
  Mail as LucideMail,
  MapPin as LucideMapPin,
  Maximize as LucideMaximize,
  Maximize2 as LucideMaximize2,
  Megaphone as LucideMegaphone,
  Menu as LucideMenu,
  MessageCircle as LucideMessageCircle,
  MessageSquare as LucideMessageSquare,
  MessageSquarePlus as LucideMessageSquarePlus,
  MessageSquareText as LucideMessageSquareText,
  MessagesSquare as LucideMessagesSquare,
  Milestone as LucideMilestone,
  Minimize2 as LucideMinimize2,
  Minus as LucideMinus,
  MinusIcon as LucideMinusIcon,
  Monitor as LucideMonitor,
  Moon as LucideMoon,
  MoreHorizontal as LucideMoreHorizontal,
  Network as LucideNetwork,
  Newspaper as LucideNewspaper,
  OctagonXIcon as LucideOctagonXIcon,
  Option as LucideOption,
  Package as LucidePackage,
  Palette as LucidePalette,
  PanelLeft as LucidePanelLeft,
  PanelLeftClose as LucidePanelLeftClose,
  PanelLeftIcon as LucidePanelLeftIcon,
  PanelLeftOpen as LucidePanelLeftOpen,
  PanelRight as LucidePanelRight,
  PanelRightClose as LucidePanelRightClose,
  PanelRightOpen as LucidePanelRightOpen,
  Paperclip as LucidePaperclip,
  Pause as LucidePause,
  PenLine as LucidePenLine,
  Pencil as LucidePencil,
  PencilLine as LucidePencilLine,
  Pin as LucidePin,
  PinOff as LucidePinOff,
  Pipette as LucidePipette,
  Play as LucidePlay,
  Plug as LucidePlug,
  PlugZap as LucidePlugZap,
  Plus as LucidePlus,
  Power as LucidePower,
  Quote as LucideQuote,
  Redo2 as LucideRedo2,
  RefreshCw as LucideRefreshCw,
  Rocket as LucideRocket,
  RotateCcw as LucideRotateCcw,
  RotateCw as LucideRotateCw,
  Rows3 as LucideRows3,
  Save as LucideSave,
  ScrollText as LucideScrollText,
  Search as LucideSearch,
  SearchIcon as LucideSearchIcon,
  Send as LucideSend,
  Server as LucideServer,
  Settings as LucideSettings,
  Settings2 as LucideSettings2,
  Shapes as LucideShapes,
  Shield as LucideShield,
  ShieldCheck as LucideShieldCheck,
  ShieldOff as LucideShieldOff,
  SignalHigh as LucideSignalHigh,
  SkipForward as LucideSkipForward,
  SlidersHorizontal as LucideSlidersHorizontal,
  Smartphone as LucideSmartphone,
  SmilePlus as LucideSmilePlus,
  Space as LucideSpace,
  Sparkles as LucideSparkles,
  Square as LucideSquare,
  SquarePen as LucideSquarePen,
  SquareTerminal as LucideSquareTerminal,
  Star as LucideStar,
  Strikethrough as LucideStrikethrough,
  Sun as LucideSun,
  Table2 as LucideTable2,
  Tablet as LucideTablet,
  Tag as LucideTag,
  Tags as LucideTags,
  Target as LucideTarget,
  Terminal as LucideTerminal,
  Trash2 as LucideTrash2,
  TriangleAlert as LucideTriangleAlert,
  TriangleAlertIcon as LucideTriangleAlertIcon,
  Type as LucideType,
  Undo2 as LucideUndo2,
  Unlink as LucideUnlink,
  Unplug as LucideUnplug,
  Upload as LucideUpload,
  User as LucideUser,
  UserMinus as LucideUserMinus,
  UserPen as LucideUserPen,
  UserPlus as LucideUserPlus,
  UserRound as LucideUserRound,
  Users as LucideUsers,
  Waves as LucideWaves,
  Webhook as LucideWebhook,
  Wifi as LucideWifi,
  WifiHigh as LucideWifiHigh,
  WifiOff as LucideWifiOff,
  Workflow as LucideWorkflow,
  WrapText as LucideWrapText,
  Wrench as LucideWrench,
  X as LucideX,
  XCircle as LucideXCircle,
  XIcon as LucideXIcon,
  Zap as LucideZap,
} from "lucide-react";

/**
 * The only module allowed to import an icon library; everything else imports
 * icons from `@multica/ui/icons`. Every glyph the product uses is listed here
 * by name and typed as {@link IconComponent}, so swapping one for a
 * Multica-drawn icon is a one-line change that reaches every surface at once.
 * Add a name here before using a new icon.
 */

export interface IconProps extends Partial<SVGProps<SVGSVGElement>> {
  size?: string | number;
}

export type IconComponent = ForwardRefExoticComponent<
  Omit<IconProps, "ref"> & RefAttributes<SVGSVGElement>
>;

export const Activity: IconComponent = LucideActivity;
export const AlarmClock: IconComponent = LucideAlarmClock;
export const AlertCircle: IconComponent = LucideAlertCircle;
export const AlertTriangle: IconComponent = LucideAlertTriangle;
export const AppWindow: IconComponent = LucideAppWindow;
export const Archive: IconComponent = LucideArchive;
export const ArchiveRestore: IconComponent = LucideArchiveRestore;
export const ArrowBigUp: IconComponent = LucideArrowBigUp;
export const ArrowDown: IconComponent = LucideArrowDown;
export const ArrowDownNarrowWide: IconComponent = LucideArrowDownNarrowWide;
export const ArrowDownToLine: IconComponent = LucideArrowDownToLine;
export const ArrowLeft: IconComponent = LucideArrowLeft;
export const ArrowLeftRight: IconComponent = LucideArrowLeftRight;
export const ArrowRight: IconComponent = LucideArrowRight;
export const ArrowRightToLine: IconComponent = LucideArrowRightToLine;
export const ArrowUp: IconComponent = LucideArrowUp;
export const ArrowUpCircle: IconComponent = LucideArrowUpCircle;
export const ArrowUpDown: IconComponent = LucideArrowUpDown;
export const ArrowUpNarrowWide: IconComponent = LucideArrowUpNarrowWide;
export const ArrowUpRight: IconComponent = LucideArrowUpRight;
export const AtSign: IconComponent = LucideAtSign;
export const Ban: IconComponent = LucideBan;
export const BarChart3: IconComponent = LucideBarChart3;
export const Bell: IconComponent = LucideBell;
export const Blocks: IconComponent = LucideBlocks;
export const Bold: IconComponent = LucideBold;
export const BookCheck: IconComponent = LucideBookCheck;
export const BookOpen: IconComponent = LucideBookOpen;
export const BookOpenText: IconComponent = LucideBookOpenText;
export const Bookmark: IconComponent = LucideBookmark;
export const Bot: IconComponent = LucideBot;
export const BotOff: IconComponent = LucideBotOff;
export const Box: IconComponent = LucideBox;
export const Brain: IconComponent = LucideBrain;
export const Briefcase: IconComponent = LucideBriefcase;
export const BriefcaseBusiness: IconComponent = LucideBriefcaseBusiness;
export const Bug: IconComponent = LucideBug;
export const Building2: IconComponent = LucideBuilding2;
export const Calendar: IconComponent = LucideCalendar;
export const CalendarClock: IconComponent = LucideCalendarClock;
export const CalendarDays: IconComponent = LucideCalendarDays;
export const Camera: IconComponent = LucideCamera;
export const ChartGantt: IconComponent = LucideChartGantt;
export const Check: IconComponent = LucideCheck;
export const CheckCheck: IconComponent = LucideCheckCheck;
export const CheckCircle2: IconComponent = LucideCheckCircle2;
export const CheckIcon: IconComponent = LucideCheckIcon;
export const ChevronDown: IconComponent = LucideChevronDown;
export const ChevronDownIcon: IconComponent = LucideChevronDownIcon;
export const ChevronLeft: IconComponent = LucideChevronLeft;
export const ChevronLeftIcon: IconComponent = LucideChevronLeftIcon;
export const ChevronRight: IconComponent = LucideChevronRight;
export const ChevronRightIcon: IconComponent = LucideChevronRightIcon;
export const ChevronUp: IconComponent = LucideChevronUp;
export const ChevronUpIcon: IconComponent = LucideChevronUpIcon;
export const ChevronsUpDown: IconComponent = LucideChevronsUpDown;
export const Circle: IconComponent = LucideCircle;
export const CircleAlert: IconComponent = LucideCircleAlert;
export const CircleCheck: IconComponent = LucideCircleCheck;
export const CircleCheckIcon: IconComponent = LucideCircleCheckIcon;
export const CircleDashed: IconComponent = LucideCircleDashed;
export const CircleDot: IconComponent = LucideCircleDot;
export const CircleEqual: IconComponent = LucideCircleEqual;
export const CircleHelp: IconComponent = LucideCircleHelp;
export const CircleOff: IconComponent = LucideCircleOff;
export const CirclePause: IconComponent = LucideCirclePause;
export const CircleSlash: IconComponent = LucideCircleSlash;
export const CircleUser: IconComponent = LucideCircleUser;
export const Clock: IconComponent = LucideClock;
export const Clock3: IconComponent = LucideClock3;
export const Cloud: IconComponent = LucideCloud;
export const Code: IconComponent = LucideCode;
export const Code2: IconComponent = LucideCode2;
export const CodeXml: IconComponent = LucideCodeXml;
export const Coins: IconComponent = LucideCoins;
export const Columns2: IconComponent = LucideColumns2;
export const Columns3: IconComponent = LucideColumns3;
export const Command: IconComponent = LucideCommand;
export const Compass: IconComponent = LucideCompass;
export const Component: IconComponent = LucideComponent;
export const Copy: IconComponent = LucideCopy;
export const CornerDownLeft: IconComponent = LucideCornerDownLeft;
export const CornerDownRight: IconComponent = LucideCornerDownRight;
export const CornerLeftUp: IconComponent = LucideCornerLeftUp;
export const CornerUpLeft: IconComponent = LucideCornerUpLeft;
export const Cpu: IconComponent = LucideCpu;
export const CreditCard: IconComponent = LucideCreditCard;
export const Crown: IconComponent = LucideCrown;
export const Database: IconComponent = LucideDatabase;
export const Delete: IconComponent = LucideDelete;
export const Dices: IconComponent = LucideDices;
export const Download: IconComponent = LucideDownload;
export const Ellipsis: IconComponent = LucideEllipsis;
export const ExternalLink: IconComponent = LucideExternalLink;
export const Eye: IconComponent = LucideEye;
export const EyeOff: IconComponent = LucideEyeOff;
export const FastForward: IconComponent = LucideFastForward;
export const File: IconComponent = LucideFile;
export const FileArchive: IconComponent = LucideFileArchive;
export const FileAudio: IconComponent = LucideFileAudio;
export const FileCode: IconComponent = LucideFileCode;
export const FileEdit: IconComponent = LucideFileEdit;
export const FileImage: IconComponent = LucideFileImage;
export const FilePen: IconComponent = LucideFilePen;
export const FilePlus: IconComponent = LucideFilePlus;
export const FilePlus2: IconComponent = LucideFilePlus2;
export const FileQuestion: IconComponent = LucideFileQuestion;
export const FileSearch: IconComponent = LucideFileSearch;
export const FileSpreadsheet: IconComponent = LucideFileSpreadsheet;
export const FileText: IconComponent = LucideFileText;
export const FileVideo: IconComponent = LucideFileVideo;
export const Filter: IconComponent = LucideFilter;
export const FilterX: IconComponent = LucideFilterX;
export const Flag: IconComponent = LucideFlag;
export const Folder: IconComponent = LucideFolder;
export const FolderGit: IconComponent = LucideFolderGit;
export const FolderGit2: IconComponent = LucideFolderGit2;
export const FolderKanban: IconComponent = LucideFolderKanban;
export const FolderMinus: IconComponent = LucideFolderMinus;
export const FolderOpen: IconComponent = LucideFolderOpen;
export const Frame: IconComponent = LucideFrame;
export const Gauge: IconComponent = LucideGauge;
export const GitBranch: IconComponent = LucideGitBranch;
export const GitMerge: IconComponent = LucideGitMerge;
export const GitPullRequest: IconComponent = LucideGitPullRequest;
export const GitPullRequestArrow: IconComponent = LucideGitPullRequestArrow;
export const GitPullRequestClosed: IconComponent = LucideGitPullRequestClosed;
export const GitPullRequestDraft: IconComponent = LucideGitPullRequestDraft;
export const Globe: IconComponent = LucideGlobe;
export const Globe2: IconComponent = LucideGlobe2;
export const GraduationCap: IconComponent = LucideGraduationCap;
export const GripVertical: IconComponent = LucideGripVertical;
export const HardDrive: IconComponent = LucideHardDrive;
export const Hash: IconComponent = LucideHash;
export const Heading1: IconComponent = LucideHeading1;
export const Heading2: IconComponent = LucideHeading2;
export const Heading3: IconComponent = LucideHeading3;
export const Heart: IconComponent = LucideHeart;
export const HelpCircle: IconComponent = LucideHelpCircle;
export const Highlighter: IconComponent = LucideHighlighter;
export const History: IconComponent = LucideHistory;
export const Hourglass: IconComponent = LucideHourglass;
export const Image: IconComponent = LucideImage;
export const ImageIcon: IconComponent = LucideImageIcon;
export const ImagePlus: IconComponent = LucideImagePlus;
export const Inbox: IconComponent = LucideInbox;
export const Info: IconComponent = LucideInfo;
export const InfoIcon: IconComponent = LucideInfoIcon;
export const Italic: IconComponent = LucideItalic;
export const KeyRound: IconComponent = LucideKeyRound;
export const Keyboard: IconComponent = LucideKeyboard;
export const Laptop: IconComponent = LucideLaptop;
export const Layers: IconComponent = LucideLayers;
export const Layers3: IconComponent = LucideLayers3;
export const LayoutDashboard: IconComponent = LucideLayoutDashboard;
export const LayoutGrid: IconComponent = LucideLayoutGrid;
export const LayoutTemplate: IconComponent = LucideLayoutTemplate;
export const Lightbulb: IconComponent = LucideLightbulb;
export const Link: IconComponent = LucideLink;
export const Link2: IconComponent = LucideLink2;
export const List: IconComponent = LucideList;
export const ListChecks: IconComponent = LucideListChecks;
export const ListChevronsDownUp: IconComponent = LucideListChevronsDownUp;
export const ListChevronsUpDown: IconComponent = LucideListChevronsUpDown;
export const ListEnd: IconComponent = LucideListEnd;
export const ListOrdered: IconComponent = LucideListOrdered;
export const ListPlus: IconComponent = LucideListPlus;
export const ListTodo: IconComponent = LucideListTodo;
export const ListTree: IconComponent = LucideListTree;
export const ListX: IconComponent = LucideListX;
export const Loader2: IconComponent = LucideLoader2;
export const Loader2Icon: IconComponent = LucideLoader2Icon;
export const LoaderCircle: IconComponent = LucideLoaderCircle;
export const LocateFixed: IconComponent = LucideLocateFixed;
export const Lock: IconComponent = LucideLock;
export const LockKeyhole: IconComponent = LucideLockKeyhole;
export const LogIn: IconComponent = LucideLogIn;
export const LogOut: IconComponent = LucideLogOut;
export const Mail: IconComponent = LucideMail;
export const MapPin: IconComponent = LucideMapPin;
export const Maximize: IconComponent = LucideMaximize;
export const Maximize2: IconComponent = LucideMaximize2;
export const Megaphone: IconComponent = LucideMegaphone;
export const Menu: IconComponent = LucideMenu;
export const MessageCircle: IconComponent = LucideMessageCircle;
export const MessageSquare: IconComponent = LucideMessageSquare;
export const MessageSquarePlus: IconComponent = LucideMessageSquarePlus;
export const MessageSquareText: IconComponent = LucideMessageSquareText;
export const MessagesSquare: IconComponent = LucideMessagesSquare;
export const Milestone: IconComponent = LucideMilestone;
export const Minimize2: IconComponent = LucideMinimize2;
export const Minus: IconComponent = LucideMinus;
export const MinusIcon: IconComponent = LucideMinusIcon;
export const Monitor: IconComponent = LucideMonitor;
export const Moon: IconComponent = LucideMoon;
export const MoreHorizontal: IconComponent = LucideMoreHorizontal;
export const Network: IconComponent = LucideNetwork;
export const Newspaper: IconComponent = LucideNewspaper;
export const OctagonXIcon: IconComponent = LucideOctagonXIcon;
export const Option: IconComponent = LucideOption;
export const Package: IconComponent = LucidePackage;
export const Palette: IconComponent = LucidePalette;
export const PanelLeft: IconComponent = LucidePanelLeft;
export const PanelLeftClose: IconComponent = LucidePanelLeftClose;
export const PanelLeftIcon: IconComponent = LucidePanelLeftIcon;
export const PanelLeftOpen: IconComponent = LucidePanelLeftOpen;
export const PanelRight: IconComponent = LucidePanelRight;
export const PanelRightClose: IconComponent = LucidePanelRightClose;
export const PanelRightOpen: IconComponent = LucidePanelRightOpen;
export const Paperclip: IconComponent = LucidePaperclip;
export const Pause: IconComponent = LucidePause;
export const PenLine: IconComponent = LucidePenLine;
export const Pencil: IconComponent = LucidePencil;
export const PencilLine: IconComponent = LucidePencilLine;
export const Pin: IconComponent = LucidePin;
export const PinOff: IconComponent = LucidePinOff;
export const Pipette: IconComponent = LucidePipette;
export const Play: IconComponent = LucidePlay;
export const Plug: IconComponent = LucidePlug;
export const PlugZap: IconComponent = LucidePlugZap;
export const Plus: IconComponent = LucidePlus;
export const Power: IconComponent = LucidePower;
export const Quote: IconComponent = LucideQuote;
export const Redo2: IconComponent = LucideRedo2;
export const RefreshCw: IconComponent = LucideRefreshCw;
export const Rocket: IconComponent = LucideRocket;
export const RotateCcw: IconComponent = LucideRotateCcw;
export const RotateCw: IconComponent = LucideRotateCw;
export const Rows3: IconComponent = LucideRows3;
export const Save: IconComponent = LucideSave;
export const ScrollText: IconComponent = LucideScrollText;
export const Search: IconComponent = LucideSearch;
export const SearchIcon: IconComponent = LucideSearchIcon;
export const Send: IconComponent = LucideSend;
export const Server: IconComponent = LucideServer;
export const Settings: IconComponent = LucideSettings;
export const Settings2: IconComponent = LucideSettings2;
export const Shapes: IconComponent = LucideShapes;
export const Shield: IconComponent = LucideShield;
export const ShieldCheck: IconComponent = LucideShieldCheck;
export const ShieldOff: IconComponent = LucideShieldOff;
export const SignalHigh: IconComponent = LucideSignalHigh;
export const SkipForward: IconComponent = LucideSkipForward;
export const SlidersHorizontal: IconComponent = LucideSlidersHorizontal;
export const Smartphone: IconComponent = LucideSmartphone;
export const SmilePlus: IconComponent = LucideSmilePlus;
export const Space: IconComponent = LucideSpace;
export const Sparkles: IconComponent = LucideSparkles;
export const Square: IconComponent = LucideSquare;
export const SquarePen: IconComponent = LucideSquarePen;
export const SquareTerminal: IconComponent = LucideSquareTerminal;
export const Star: IconComponent = LucideStar;
export const Strikethrough: IconComponent = LucideStrikethrough;
export const Sun: IconComponent = LucideSun;
export const Table2: IconComponent = LucideTable2;
export const Tablet: IconComponent = LucideTablet;
export const Tag: IconComponent = LucideTag;
export const Tags: IconComponent = LucideTags;
export const Target: IconComponent = LucideTarget;
export const Terminal: IconComponent = LucideTerminal;
export const Trash2: IconComponent = LucideTrash2;
export const TriangleAlert: IconComponent = LucideTriangleAlert;
export const TriangleAlertIcon: IconComponent = LucideTriangleAlertIcon;
export const Type: IconComponent = LucideType;
export const Undo2: IconComponent = LucideUndo2;
export const Unlink: IconComponent = LucideUnlink;
export const Unplug: IconComponent = LucideUnplug;
export const Upload: IconComponent = LucideUpload;
export const User: IconComponent = LucideUser;
export const UserMinus: IconComponent = LucideUserMinus;
export const UserPen: IconComponent = LucideUserPen;
export const UserPlus: IconComponent = LucideUserPlus;
export const UserRound: IconComponent = LucideUserRound;
export const Users: IconComponent = LucideUsers;
export const Waves: IconComponent = LucideWaves;
export const Webhook: IconComponent = LucideWebhook;
export const Wifi: IconComponent = LucideWifi;
export const WifiHigh: IconComponent = LucideWifiHigh;
export const WifiOff: IconComponent = LucideWifiOff;
export const Workflow: IconComponent = LucideWorkflow;
export const WrapText: IconComponent = LucideWrapText;
export const Wrench: IconComponent = LucideWrench;
export const X: IconComponent = LucideX;
export const XCircle: IconComponent = LucideXCircle;
export const XIcon: IconComponent = LucideXIcon;
export const Zap: IconComponent = LucideZap;
