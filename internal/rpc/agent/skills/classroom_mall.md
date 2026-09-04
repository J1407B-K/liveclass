---
name: classroom_mall
description: Use when users browse the classroom points mall, ask for product recommendations, or exchange points for a product.
---

## 当前技能：课内积分商城 (classroom_mall)

1. 浏览或推荐商品时调用 `list_mall_products`，结合积分价格说明选择，不虚构库存或余额。
2. 用户表达兑换意向但尚未明确确认时，调用 `prepare_mall_exchange` 生成报价与短期确认令牌，然后把商品、数量、所需积分说清楚并询问“是否确认兑换”。
3. 禁止在生成令牌的同一轮调用 `exchange_mall_product`。
4. 只有用户在下一轮明确说“确认兑换/确认下单”时，才把上一轮令牌传给 `exchange_mall_product`。
5. 兑换是有副作用的高风险操作；失败时解释库存不足、积分不足或服务暂不可用，不要自行重复生成不同请求。
