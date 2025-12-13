<?php

namespace AutoUnlock;

use EDACerton\PluginUtils\Translator;

/*
    Copyright (C) 2025  Derek Kaser

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

if ( ! defined(__NAMESPACE__ . '\PLUGIN_ROOT') || ! defined(__NAMESPACE__ . '\PLUGIN_NAME')) {
    throw new \RuntimeException("Common file not loaded.");
}

$tr           = $tr ?? new Translator(PLUGIN_ROOT);
$csrfToken    = Utils::getCsrfToken();
$arrayStopped = Utils::isArrayStopped();
?>

<div class="output_display" style="display:none;">
    <div class="share_instructions" style="display: none;">
        <p><?= $tr->tr("share_instructions"); ?></p>
        <p><strong><?= $tr->tr("not_shown_again"); ?></strong></p>
    </div>
    <pre id="command_output"></pre>
    <input type="button" id="continue_button" disabled onclick="location.reload()" value="<?= $tr->tr("continue"); ?>" />
</div>

<script type="text/javascript">
    async function streamToOutput(response, outputId) {
        const reader = response.body.getReader();
        let decoder = new TextDecoder();
        let output = '';
        document.getElementById(outputId).textContent = '';
        while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            const chunk = decoder.decode(value, { stream: true });
            output += chunk;
            document.getElementById(outputId).textContent = output;
        }
    }
</script>

<?php
// Show initialize section if /boot/config/plugins/auto-unlock/state.json or /boot/config/plugins/auto-unlock/unlock.enc do not exist
if ( ! file_exists(Utils::STATE_FILE) && ! file_exists(Utils::ENC_FILE)) {
    ?>

<script type="text/javascript">
    function clearKeyfile() {
        document.getElementById('keyfile').value = "";
        document.getElementById('clear_file').disabled = true;
        document.getElementById('passphrase').disabled = false;
        verifyInitializeInputs();
    }

    function setKeyfile() {
        document.getElementById('clear_file').disabled = false;
        document.getElementById('passphrase').value = "";
        document.getElementById('passphrase').disabled = true;
        verifyInitializeInputs();
    }

    function initialize() {
        const sharesTotal   = document.getElementById('shares_total').value;
        const sharesUnlock  = document.getElementById('shares_unlock').value;
        const passphrase    = document.getElementById('passphrase').value;
        const keyfileInput  = document.getElementById('keyfile');
        let keyfileContent  = null;

        // Hide input form and show result area
        document.querySelector('.initialize_form').style.display = 'none';
        document.querySelector('.output_display').style.display = 'block';
        document.getElementById('command_output').textContent = 'Initializing... Please wait.';

        if (keyfileInput.files.length > 0) {
            const fileReader = new FileReader();
            fileReader.onload = function(event) {
                keyfileContent = event.target.result;
                submitInitialization(sharesTotal, sharesUnlock, keyfileContent);
            };
            fileReader.readAsDataURL(keyfileInput.files[0]);
        } else {
            keyfileContent = window.btoa(unescape(encodeURIComponent(passphrase)));
            submitInitialization(sharesTotal, sharesUnlock, keyfileContent);
        }
    }

    async function submitInitialization(sharesTotal, sharesUnlock, keyfileContent) {
        const formData = new URLSearchParams({
            'shares_total': sharesTotal,
            'shares_unlock': sharesUnlock,
            'keyfile_data': keyfileContent,
            'csrf_token': '<?= $csrfToken; ?>'
        });

        try {
            const response = await fetch('/plugins/auto-unlock/action.php/initialize', {
                method: 'POST',
                body: formData,
                signal: AbortSignal.timeout(30000)
            });
            await streamToOutput(response, 'command_output');
            document.querySelector('.share_instructions').style.display = 'block';
            document.getElementById('continue_button').disabled = false;
        } catch (error) {
            document.getElementById('command_output').textContent = 'Error during initialization: ' + error.message;
            document.getElementById('continue_button').disabled = false;
        }
    }

    function verifyInitializeInputs() {
        const sharesTotal   = document.getElementById('shares_total').value;
        const sharesUnlock  = document.getElementById('shares_unlock').value;
        const passphrase    = document.getElementById('passphrase').value;
        const keyfileInput  = document.getElementById('keyfile');

        let isValid = true;

        if (sharesTotal < 1 || sharesTotal > 100) {
            isValid = false;
        }

        if (sharesUnlock < 1 || sharesUnlock > 100 || sharesUnlock > sharesTotal) {
            isValid = false;
        }

        if (passphrase.length === 0 && keyfileInput.files.length === 0) {
            isValid = false;
        }

        document.getElementById('initialize').disabled = !isValid;
    }
</script>
<div class="initialize_form">
<table class="unraid tablesorter"><thead><tr><td><?= $tr->tr("initialize"); ?></td></tr></thead></table>

    <input type="hidden" name="file" value="">
    <dl>
        <dt><?= $tr->tr("shares_total"); ?></dt>
        <dd>
            <input type="number" id="shares_total" name="shares_total" value="5" min="1" max="100" oninput="verifyInitializeInputs()" />
        </dd>
    </dl>
    <dl>
        <dt><?= $tr->tr("shares_unlock"); ?></dt>
        <dd>
            <input type="number" id="shares_unlock" name="shares_unlock" value="3" min="1" max="100" oninput="verifyInitializeInputs()" />
        </dd>
    </dl>
    <dl>
        <dt><?= $tr->tr("passphrase"); ?></dt>
        <dd>
            <input type="password" id="passphrase" name="passphrase" value="" oninput="verifyInitializeInputs()" />
        </dd>
    </dl>
    <dl>
        <dt><?= $tr->tr("keyfile"); ?></dt>
        <dd>
            <input type="file" id="keyfile" name="keyfile" value="" onchange="setKeyfile()" />
            <input type="button" id="clear_file" name="clear_file" value="<?= $tr->tr("clear"); ?>" disabled onclick="clearKeyfile()" />
        </dd>
    </dl>
    <dl>
        <dt><?= $tr->tr("initialize"); ?></dt>
        <dd>
            <input type="button" id="initialize" name="initialize" disabled onclick="initialize()" value="<?= $tr->tr("initialize"); ?>" />
        </dd>
    </dl>
</div>

<?php } else {
    $text = file_exists(Utils::CONFIG_FILE) ? file_get_contents(Utils::CONFIG_FILE) : '';
    ?>
<div class="config_forms">
<script type="text/javascript">
    async function testPath() {
        // POST to /plugins/auto-unlock/action.php/test_path with test_path parameter
        const testPathValue = document.getElementById('test_path').value;

        document.querySelector('.output_display').style.display = 'block';
        document.querySelector('.config_forms').style.display = 'none';
        document.getElementById('command_output').textContent = 'Testing path... Please wait.';

        const formData = new URLSearchParams({
            'test_path': testPathValue,
            'csrf_token': '<?= $csrfToken; ?>'
        });

        try {
            const response = await fetch('/plugins/auto-unlock/action.php/test_path', {
                method: 'POST',
                body: formData,
                signal: AbortSignal.timeout(20000)
            });
            await streamToOutput(response, 'command_output');
        } catch (error) {
            document.getElementById('command_output').textContent = 'Error during path test: ' + error.message;
        }
        document.getElementById('continue_button').disabled = false;
    }

    async function obscure() {
        const obscureValue = document.getElementById('obscure_value').value;

        const formData = new URLSearchParams({
            'obscure_value': obscureValue,
            'csrf_token': '<?= $csrfToken; ?>'
        });

        try {
            const response = await fetch('/plugins/auto-unlock/action.php/obscure', {
                method: 'POST',
                body: formData,
                signal: AbortSignal.timeout(10000)
            });

            const result = await response.text();
            document.getElementById('obscure_output').textContent = result;
        } catch (error) {
            document.getElementById('obscure_output').textContent = 'Error during obscuring value: ' + error.message;
        }
    }

    async function testConfig() {
        const formData = new URLSearchParams({
            'csrf_token': '<?= $csrfToken; ?>'
        });

        document.querySelector('.output_display').style.display = 'block';
        document.querySelector('.config_forms').style.display = 'none';
        document.getElementById('command_output').textContent = "Testing configuration... Please wait.";

        try {
            const response = await fetch('/plugins/auto-unlock/action.php/test', {
                method: 'POST',
                body: formData,
                signal: AbortSignal.timeout(60000)
            });
            await streamToOutput(response, 'command_output');
        } catch (error) {
            document.getElementById('command_output').textContent = 'Error during configuration test: ' + error.message;
        }
        document.getElementById('continue_button').disabled = false;
    }

    async function unlockArray() {
        const formData = new URLSearchParams({
            'csrf_token': '<?= $csrfToken; ?>'
        });

        document.querySelector('.output_display').style.display = 'block';
        document.querySelector('.config_forms').style.display = 'none';
        document.getElementById('command_output').textContent = "Unlocking array... Please wait.";

        try {
            const response = await fetch('/plugins/auto-unlock/action.php/open', {
                method: 'POST',
                body: formData,
                signal: AbortSignal.timeout(120000)
            });
            await streamToOutput(response, 'command_output');
        } catch (error) {
            document.getElementById('command_output').textContent = 'Error during array unlock: ' + error.message;
        }
        document.getElementById('continue_button').disabled = false;
    }
</script>

<table class="unraid tablesorter"><thead><tr><td><?= $tr->tr("download_locations"); ?></td></tr></thead></table>
<p><?= $tr->tr("download_locations_instructions"); ?></p>
<pre>
<?php require PLUGIN_ROOT . "/sample-locations.txt"; ?>
</pre>
<form method="post" action="/update.php" target="progressFrame">
	<input type="hidden" name="#include" value="/webGui/include/update.file.php">
	<input type="hidden" name="#raw_file" value="true">
	<input type="hidden" name="#file" value="<?= Utils::CONFIG_FILE; ?>">

    <dl>
        <dt><?= $tr->tr('download_locations'); ?></dt>
        <dd>
            <textarea spellcheck="false" wrap="off" rows="10" name="text" style="font-family:bitstream;"><?= htmlspecialchars($text ?: "");?></textarea>
        </dd>
    </dl>

    <dl><dt>&nbsp;</dt><dd>
	<span><input type="submit" value='<?= $tr->tr('apply'); ?>'><input type="button" value="<?= $tr->tr('done'); ?>" onclick="done()"></span>
	</dd></dl>

</form>

<table class="unraid tablesorter"><thead><tr><td><?= $tr->tr("unlock_array"); ?></td></tr></thead></table>
<dl>
    <dt><?= $tr->tr("unlock_array"); ?></dt>
    <dd>
        <input type="button" id="unlock_array_button" name="unlock_array_button" value="<?= $tr->tr("unlock_array"); ?>" onclick="unlockArray()" <?= $arrayStopped ? '' : 'disabled'; ?> />
    </dd>
</dl>

<table class="unraid tablesorter"><thead><tr><td><?= $tr->tr("test_path"); ?></td></tr></thead></table>
<dl>
    <dt><?= $tr->tr("test_path"); ?></dt>
    <dd>
        <input type="text" id="test_path" name="test_path" value="" />
        <input type="button" id="test_path_button" name="test_path_button" value="<?= $tr->tr("test"); ?>" onclick="testPath()" />
    </dd>
</dl>

<table class="unraid tablesorter"><thead><tr><td><?= $tr->tr("test_configuration"); ?></td></tr></thead></table>
<dl>
    <dt><?= $tr->tr("test_configuration"); ?></dt>
    <dd>
        <input type="button" id="test_config_button" name="test_config_button" value="<?= $tr->tr("test"); ?>" onclick="testConfig()" />
    </dd>
</dl>

<table class="unraid tablesorter"><thead><tr><td><?= $tr->tr("obscure_value"); ?></td></tr></thead></table>
<dl>
    <dt><?= $tr->tr("obscure_value"); ?></dt>
    <dd>
        <input type="password" id="obscure_value" name="obscure_value" value="" />
        <input type="button" id="obscure_value_button" name="obscure_value_button" value="<?= $tr->tr("obscure"); ?>" onclick="obscure()" />
        <span style="font-family: monospace;" id="obscure_output"></span>
    </dd>
</dl>

<table class="unraid tablesorter"><thead><tr><td><?= $tr->tr("remove"); ?></td></tr></thead></table>
<form method="post" action="/plugins/auto-unlock/action.php/remove" id="remove_form" name="remove_form">
<dl>
    <dt><?= $tr->tr("remove"); ?></dt>
    <dd>
        <input type="button" id="remove_config" name="remove_config" value="<?= $tr->tr("remove"); ?>" onclick="document.getElementById('confirm_remove').disabled = false; this.disabled = true;" />
        <input type="submit" id="confirm_remove" name="confirm_remove" value="<?= $tr->tr("confirm"); ?>" disabled />
    </dd>
</dl>
</form>
</div>
<?php } ?>