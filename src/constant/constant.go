package constant

/* constants for linkedlist */
const ITERATION_DIRECTION_INORDER = 1
const ITERATION_DIRECTION_REVERSE_ORDER = -1

/* constans for zskiplist */
const ZSKIPLIST_MAXLEVEL = 64
const ZSKIPLIST_P = 0.25
const ZSKIPLIST_RANDOM_MAXLEVEL = 0xFFFF * ZSKIPLIST_P // 16383