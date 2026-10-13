import { expect, test } from '@playwright/test'

for (const platform of ['windows', 'macos'] as const) {
  for (const theme of ['light', 'dark'] as const) {
    test(`${platform} ${theme} main view`, async ({ page }) => {
      await page.goto(`/?platform=${platform}&theme=${theme}`)
      await expect(page.getByRole('heading', { name: 'Clipare' })).toBeVisible()
      await expect(page.getByText('Синхронизация включена', { exact: true })).toBeVisible()
      await expect(page.getByText('MacBook Air', { exact: true }).first()).toBeVisible()
      await expect(page).toHaveScreenshot(`${platform}-${theme}-home.png`, { fullPage: true })
    })
  }

  test(`${platform} empty devices and discovery`, async ({ page }) => {
    await page.goto(`/?platform=${platform}&scenario=empty`)
    await expect(page.getByRole('button', { name: 'Добавить устройство' })).toHaveCount(1)
    await expect(page.getByRole('button', { name: /Удалить/ })).toHaveCount(0)
    await expect(page).toHaveScreenshot(`${platform}-empty.png`, { fullPage: true })
    await page.getByRole('button', { name: 'Добавить устройство' }).click()
    await expect(page.getByText('Найденные устройства')).toHaveCount(0)
    await expect(page.getByText('Desktop-PC', { exact: true }).last()).toBeVisible()
    await expect(page.getByRole('button', { name: 'Подключить' }).first()).toBeEnabled()
    await expect(page).toHaveScreenshot(`${platform}-discovery.png`, { fullPage: true })
  })

  test(`${platform} offline, pause and connection error`, async ({ page }) => {
    await page.goto(`/?platform=${platform}&scenario=offline`)
    await expect(page.getByText('Не в сети')).toHaveCount(2)
    await page.getByRole('button', { name: 'Пауза' }).click()
    await expect(page.getByText('Синхронизация приостановлена', { exact: true })).toBeVisible()
    await page.getByLabel('Сценарий предпросмотра').selectOption('error')
    await expect(page.getByText('Ожидание NetBird')).toBeVisible()
    await expect(page).toHaveScreenshot(`${platform}-error.png`, { fullPage: true })
  })

  test(`${platform} pairing and history secrets`, async ({ page }) => {
    await page.goto(`/?platform=${platform}&scenario=pairing`)
    await expect(page.getByText('482 731')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Разрешить' })).toBeVisible()
    await page.getByRole('button', { name: 'Разрешить' }).click()
    await expect(page.getByText('Устройство подключено')).toBeVisible()
    await page.getByRole('button', { name: 'История буфера обмена' }).click()
    await expect(page.getByText('•••• •••• •••• ••••')).toBeVisible()
    await page.getByRole('button', { name: 'Показать значение' }).click()
    await expect(page.getByText('clipare-preview-secret-value')).toBeVisible()
    await page.getByRole('button', { name: 'Скрыть значение' }).click()
    await expect(page.getByText('clipare-preview-secret-value')).toHaveCount(0)
    await page.getByRole('button', { name: 'Развернуть запись' }).click()
    await expect(page.getByRole('button', { name: 'Свернуть запись' })).toBeVisible()
    await expect(page).toHaveScreenshot(`${platform}-history.png`, { fullPage: true })
  })

  test(`${platform} settings and available update`, async ({ page }) => {
    await page.goto(`/?platform=${platform}&scenario=update`)
    await page.getByRole('button', { name: 'Дополнительно' }).click()
    await expect(page.getByLabel('Канал обновлений')).toBeVisible()
    await expect(page.getByText('Авто-проверка обновлений')).toBeVisible()
    await page.locator('#advanced-settings').getByRole('button', { name: 'Расширенные параметры' }).click()
    await expect(page.getByLabel('Device ID', { exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Добавить по коду' })).toBeDisabled()
    await expect(page.getByRole('button', { name: 'Применить' })).toHaveCount(0)
    await page.getByRole('button', { name: 'Проверить обновления' }).click()
    await expect(page.getByText('Доступна версия 0.6.0-beta.5')).toBeVisible()
    await expect(page).toHaveScreenshot(`${platform}-update.png`, { fullPage: true })
  })
}
