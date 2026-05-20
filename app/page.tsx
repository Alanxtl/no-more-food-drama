export default function HomePage() {
  return (
    <main className="min-h-screen bg-paper text-ink">
      <section className="mx-auto flex min-h-screen w-full max-w-md flex-col justify-center px-5 py-8">
        <p className="text-sm text-neutral-600">no-more-food-drama</p>
        <h1 className="mt-2 text-4xl font-bold leading-tight">让你选你又不选</h1>
        <p className="mt-4 text-base leading-7 text-neutral-700">
          先把附近餐厅找出来，再让两个人各自筛掉今天不想吃的类型。
        </p>
      </section>
    </main>
  );
}
