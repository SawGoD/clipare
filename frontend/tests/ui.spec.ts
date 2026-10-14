import { expect, test } from '@playwright/test'

async function visual(page: import('@playwright/test').Page, name: string) {
  if (process.env.CLIPARE_SKIP_VISUAL !== '1') {
    await expect(page).toHaveScreenshot(name, { fullPage: true, maxDiffPixelRatio: 0.003 })
  }
}

for (const platform of ['windows', 'macos'] as const) {
  for (const theme of ['light', 'dark'] as const) {
    test(`${platform} ${theme} main view`, async ({ page }) => {
      await page.goto(`/?platform=${platform}&theme=${theme}`)
      await expect(page.getByRole('heading', { name: 'Clipare' })).toBeVisible()
      await expect(page.getByText('Синхронизация включена', { exact: true })).toBeVisible()
      await expect(page.getByText('MacBook Air', { exact: true }).first()).toBeVisible()
      await visual(page, `${platform}-${theme}-home.png`)
    })
  }

  test(`${platform} empty devices and discovery`, async ({ page }) => {
    await page.goto(`/?platform=${platform}&scenario=empty`)
    await expect(page.getByRole('button', { name: 'Добавить устройство' })).toHaveCount(1)
    await expect(page.getByRole('button', { name: /Удалить/ })).toHaveCount(0)
    await visual(page, `${platform}-empty.png`)
    await page.getByRole('button', { name: 'Добавить устройство' }).click()
    await expect(page.getByText('Найденные устройства')).toHaveCount(0)
    await expect(page.getByText('Desktop-PC', { exact: true }).last()).toBeVisible()
    await expect(page.getByRole('button', { name: 'Подключить' }).first()).toBeEnabled()
    await visual(page, `${platform}-discovery.png`)
  })

  test(`${platform} offline, pause and connection error`, async ({ page }) => {
    await page.goto(`/?platform=${platform}&scenario=offline`)
    await expect(page.getByText('Не в сети')).toHaveCount(2)
    await page.getByRole('button', { name: 'Пауза' }).click()
    await expect(page.getByText('Синхронизация приостановлена', { exact: true })).toBeVisible()
    await page.getByLabel('Сценарий предпросмотра').selectOption('error')
    await expect(page.getByText('Ожидание NetBird')).toBeVisible()
    await visual(page, `${platform}-error.png`)
  })

  test(`${platform} pairing and history secrets`, async ({ page }) => {
    await page.goto(`/?platform=${platform}&scenario=pairing`)
    await expect(page.getByText('482 731')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Разрешить' })).toBeVisible()
    await page.getByRole('button', { name: 'Разрешить' }).click()
    await expect(page.getByText('Устройство подключено')).toBeVisible()
    await expect(page.locator('.device-platform[data-platform=unknown] svg')).toBeVisible()
    await page.getByRole('button', { name: 'История буфера обмена' }).click()
    await expect(page.getByText('•••• •••• •••• ••••')).toBeVisible()
    await page.getByRole('button', { name: 'Показать значение' }).click()
    await expect(page.getByText('clipare-preview-secret-value')).toBeVisible()
    await page.getByRole('button', { name: 'Скрыть значение' }).click()
    await expect(page.getByText('clipare-preview-secret-value')).toHaveCount(0)
    await page.getByRole('button', { name: 'Развернуть запись' }).click()
    await expect(page.getByRole('button', { name: 'Свернуть запись' })).toBeVisible()
    await visual(page, `${platform}-history.png`)
  })

  test(`${platform} discovery flows into pairing and returns`, async ({ page }) => {
    await page.goto(`/?platform=${platform}&scenario=empty`)
    await page.getByRole('button', { name: 'Добавить устройство' }).click()
    await page.getByRole('button', { name: 'Подключить' }).first().click()
    await expect(page.getByText('482 731')).toBeVisible()
    await page.getByRole('button', { name: 'Отменить' }).click()
    await expect(page.getByRole('heading', { name: 'Устройства' })).toBeVisible()
  })

  test(`${platform} settings and available update`, async ({ page }) => {
    await page.goto(`/?platform=${platform}&scenario=update`)
    await page.getByRole('button', { name: 'Дополнительно' }).click()
    await expect(page.getByLabel('Канал обновлений')).toBeVisible()
    await expect(page.getByText('Авто-проверка обновлений')).toBeVisible()
    await page.getByRole('button', { name: 'Расширенные параметры' }).click()
    await expect(page.getByLabel('Device ID', { exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Добавить по коду' })).toBeDisabled()
    await expect(page.getByRole('button', { name: 'Применить' })).toHaveCount(0)
    await page.getByRole('button', { name: 'Проверить обновления' }).click()
    await expect(page.getByText('Доступна версия 0.6.0-beta.5')).toBeVisible()
    await visual(page, `${platform}-update.png`)
  })

  test(`${platform} settings icon toggles advanced controls`, async ({ page }) => {
    await page.goto(`/?platform=${platform}&theme=dark`)
    const gear = page.getByRole('button', { name: 'Расширенные параметры' })
    await expect(gear).toHaveCount(1)
    await expect(gear).toHaveAttribute('aria-expanded', 'false')
    await expect(page.locator('#advanced-settings')).toHaveCount(0)
    await gear.click()
    const advanced = page.locator('#advanced-settings')
    await expect(gear).toHaveAttribute('aria-expanded', 'true')
    await expect(advanced.getByLabel('Device ID', { exact: true })).toBeVisible()
    await gear.click()
    await expect(gear).toHaveAttribute('aria-expanded', 'false')
    await expect(advanced).toHaveCount(0)
  })

  test(`${platform} device platforms and copy feedback`, async ({ page }) => {
    await page.goto(`/?platform=${platform}`)
    await expect(page.locator('.device-platform[data-platform=macos] svg')).toBeVisible()
    await expect(page.locator('.device-platform[data-platform=windows] svg')).toBeVisible()
    await page.getByRole('button', { name: 'История буфера обмена' }).click()
    await page.getByRole('button', { name: 'Копировать запись' }).first().click()
    await expect(page.getByRole('button', { name: 'Скопировано' })).toBeVisible()
  })
}

test('Windows dropdowns remain interactive inside additional settings', async ({ page }) => {
  const errors: string[] = []
  page.on('pageerror', error => errors.push(error.message))
  await page.goto('/?platform=windows&theme=dark')
  await page.getByRole('button', { name: 'Дополнительно' }).click()
  await page.getByRole('combobox', { name: 'Канал обновлений' }).click()
  await page.getByRole('option', { name: 'Стабильные' }).click()
  await expect(page.getByRole('heading', { name: 'Clipare' })).toBeVisible()
  await page.getByRole('combobox', { name: 'Хранить' }).click()
  await page.getByRole('option', { name: '1 час' }).click()
  await expect(page.getByRole('heading', { name: 'Clipare' })).toBeVisible()
  expect(errors).toEqual([])
})
